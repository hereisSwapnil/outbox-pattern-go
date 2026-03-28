<h1>📬 Go Outbox Pattern</h1>

<p>
  <b>Reference Implementation: Bulletproof Event Publishing with Postgres Logical Replication</b>
</p>

<p>
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8.svg" alt="Go 1.21+">
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791.svg" alt="Postgres 16">
</p>

---

## 📖 The Story: The Dual-Write Nightmare

It’s a classic microservices nightmare: A user buys a Macbook. You need to save the order to your Postgres database *and* publish an `order.created` event to Kafka so the shipping service can pick it up.

You write the order to the database, but right before you publish to Kafka, your server crashes. The user was charged, the order is in the database, but shipping never hears about it. 

If you reverse the order (Kafka first, then database), Kafka might get the message but the database insert fails. Now shipping ships a Macbook that doesn't exist in your database. 

**This repository is the fix.** By using the **Outbox Pattern**, we use the ultimate source of truth—the database's Write-Ahead Log (WAL)—to guarantee that if an order is saved, the event is *always* published. It transforms a risky "dual-write" guessing game into guaranteed, bulletproof event delivery.

> [!NOTE]
> Read more about the classic pattern here: [Transactional Outbox (microservices.io)](https://microservices.io/patterns/data/transactional-outbox.html)

---

## 🏗️ The Core Rule: Atomic Writes, Decoupled Reads

The breakthrough is a simple separation of concerns: **Never publish directly to the broker from the API.**

Instead, the API is only allowed to talk to the database. It inserts the `Order` and the `OutboxEvent` in a **single, atomic database transaction**. A separate, background "Relay" service connects directly to the database's WAL to stream those events to the broker.

> [!NOTE]
> The beauty of this pattern is that the application doesn't need to know Kafka exists. It just writes to a table. The database handles the durability, and the relay handles the delivery.

---

## 🛡️ How We Guarantee Delivery

This implementation relies on three mechanical differences from a standard polling or dual-write setup:

**1. Absolute Atomicity:**
The business entity (`orders`) and the event intent (`outbox_events`) are committed in the exact same Postgres transaction. Either both succeed, or neither do. There is no middle ground.

**2. Event Streaming (Not Polling):**
We avert inefficient `SELECT * FROM outbox_events WHERE processed = false` polling loops by leveraging PostgreSQL's native logical replication (`wal_level=logical`). The database acts as a real-time event stream, pushing row-level changes the millisecond they commit.

**3. Total Decoupling:**
The background `Relay` service is completely independent of the main `App` service. It uses the `pglogrepl` library to listen to the replication slot, parse the binary WAL data, publish to the downstream broker, and strictly acknowledge the LSN back to Postgres.

---

## 🚀 Quick Start

### 1. Prerequisites
- Docker & Docker Compose
- `just` task runner (Optional, but recommended)

### 2. Start the Stack
Spin up the PostgreSQL database, the App API, and the WAL Relay service all at once:
```bash
just start
```
*(If you don't have `just`, run `docker-compose up -d --build`)*

### 3. Create an Order
Send a request to the API to create an order. The API will insert the order and the outbox event atomically.
```bash
just test-order item="Macbook"
```

### 4. Watch the Relay Catch It
Watch the logs of the decoupled Relay service. You will see it receive the event from the Postgres WAL in real-time:
```bash
just logs
```
> `📨 Event received → type: order.created | payload: {"item": "item=Macbook"}`

---

## ⚙️ Configuration & Under the Hood

The magic happens in the database configuration defined in `scripts/init.sql`:

- **Replication Slot:** We create `outbox_slot` using `pg_create_logical_replication_slot`. This keeps track of exactly how far the Relay has read in the log.
- **Publication:** We create `outbox_pub` specifically for the `outbox_events` table. This acts as a filter so the Relay doesn't receive WAL data for standard `orders` inserts.

---

## 🔒 Threat Model & Gotchas (Assumptions)

1. **Replication Slot Persistence (The Disk Threat):** If the Relay goes offline, Postgres will hold onto the WAL files forever until the Relay comes back and acknowledges them. This can fill up your disk! Always monitor replication slot lag in production.
2. **Schema Evolution:** `pglogrepl` requires handling `RelationMessage` to understand table schema changes. This demo uses a simplified column index mapping for brevity.
3. **Bring Your Own Broker:** The current `Relay` prints to `stdout`. To make this production-ready, swap the `fmt.Printf` in `cmd/relay/main.go` with your actual message broker publish method (like `kafka.Publish()`).

---

## 📁 Project Structure

```
outbox-pattern-go/
├── cmd/
│   ├── app/           # The main HTTP API to create orders
│   └── relay/         # The WAL relay service (Postgres WAL -> Broker)
├── internal/
│   ├── domain/        # Business models (Order, OutboxEvent)
│   ├── repository/    # Postgres tx & insert logic
│   ├── service/       # Use cases (Order creation orchestrating the tx)
│   └── relay/         # Postgres logical replication consumer logic
├── pkg/
│   └── db/            # Database connection logic with retries
├── scripts/
│   └── init.sql       # Postgres schema & replication slot setup
├── Dockerfile         # Multi-target build for App & Relay
├── docker-compose.yml # Orchestrator
└── Justfile           # Task runner commands
```
CREATE TABLE orders (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item       TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE outbox_events (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_type TEXT NOT NULL,
  payload    JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Create the replication slot
SELECT pg_create_logical_replication_slot('outbox_slot', 'pgoutput');

-- Create the publication for just the outbox table
CREATE PUBLICATION outbox_pub FOR TABLE outbox_events;

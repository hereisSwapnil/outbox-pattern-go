package domain

import "time"

type Order struct {
	ID        string    `json:"id"`
	Item      string    `json:"item"`
	CreatedAt time.Time `json:"created_at"`
}

type OutboxEvent struct {
	ID        string    `json:"id"`
	EventType string    `json:"event_type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

package models

import "time"

type OutboxEvent struct {
	ID          int64
	Topic       string
	AggregateID int64
	Payload     []byte
	OccurredAt  time.Time
}

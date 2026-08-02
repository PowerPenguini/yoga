package validators

import (
	"testing"
	"time"

	"main/models"
)

func TestOutboxEventValidatorValidate(t *testing.T) {
	event := &models.OutboxEvent{
		Topic:       "item.created",
		AggregateID: 42,
		Payload:     []byte(`{"code":"ITEM-01"}`),
		OccurredAt:  time.Now(),
	}
	if err := NewOutboxEventValidator().Validate(event); err != nil {
		t.Fatalf("valid outbox event: %v", err)
	}
}

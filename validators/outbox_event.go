package validators

import (
	"encoding/json"
	"strings"

	"github.com/PowerPenguini/errs"

	"main/models"
)

type OutboxEventValidator struct{}

func NewOutboxEventValidator() *OutboxEventValidator {
	return &OutboxEventValidator{}
}

func (v *OutboxEventValidator) Validate(event *models.OutboxEvent) error {
	validationErrors := errs.NewErrorList()
	if event == nil {
		return errs.NewError("outbox_event_required", "outbox event is required", errs.ValidationType, nil)
	}
	if strings.TrimSpace(event.Topic) == "" {
		validationErrors.Append(errs.NewFieldError("outbox_topic_required", "topic", "topic is required", errs.ValidationType, nil))
	}
	if event.AggregateID <= 0 {
		validationErrors.Append(errs.NewFieldError("outbox_aggregate_id_required", "aggregate_id", "aggregate id is required", errs.ValidationType, nil))
	}
	if len(event.Payload) == 0 || !json.Valid(event.Payload) {
		validationErrors.Append(errs.NewFieldError("outbox_payload_invalid", "payload", "payload must be valid JSON", errs.ValidationType, nil))
	}
	if event.OccurredAt.IsZero() {
		validationErrors.Append(errs.NewFieldError("outbox_occurred_at_required", "occurred_at", "occurred at is required", errs.ValidationType, nil))
	}
	if validationErrors.Len() > 0 {
		return validationErrors
	}
	return nil
}

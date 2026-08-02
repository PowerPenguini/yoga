package logic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PowerPenguini/errs"

	"main/di"
	"main/models"
)

const (
	ItemCreatedTopic  = "item.created"
	ItemUpdatedTopic  = "item.updated"
	ItemArchivedTopic = "item.archived"
)

type EnqueueItemEvent struct {
	Topic  string
	ItemID int64
	Data   map[string]any
}

func (e *EnqueueItemEvent) Normalize() {
	e.Topic = strings.ToLower(strings.TrimSpace(e.Topic))
}

func (e *EnqueueItemEvent) Validate() error {
	if e.ItemID <= 0 {
		return errs.NewFieldError("item_event_id_required", "item_id", "item id is required", errs.ValidationType, nil)
	}
	switch e.Topic {
	case ItemCreatedTopic, ItemUpdatedTopic, ItemArchivedTopic:
		return nil
	default:
		return errs.NewFieldError("item_event_topic_invalid", "topic", "unsupported item event topic", errs.ValidationType, nil)
	}
}

func (e *EnqueueItemEvent) Execute(ctx context.Context, deps *di.DI) error {
	return di.ExecuteInTxNoResult(deps, func(txDI *di.DI) error {
		e.Normalize()
		if err := e.Validate(); err != nil {
			return err
		}

		payload, err := json.Marshal(e.Data)
		if err != nil {
			return errs.NewError("item_event_encode_failed", "failed to encode item event", errs.InternalType, err)
		}
		event := &models.OutboxEvent{
			Topic:       e.Topic,
			AggregateID: e.ItemID,
			Payload:     payload,
			OccurredAt:  txDI.Clock.Now(),
		}
		if err := txDI.OutboxEventValidator.Validate(event); err != nil {
			return err
		}
		id, err := txDI.OutboxEventRepo.Insert(ctx, event)
		if err != nil {
			return errs.NewError("item_event_enqueue_failed", "failed to enqueue item event", errs.InternalType, err)
		}
		event.ID = id
		return nil
	})
}

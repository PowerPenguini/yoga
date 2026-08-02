package repos

import (
	"context"

	"main/models"
)

type OutboxEventRepo struct {
	db DBTX
}

func NewOutboxEventRepo(db DBTX) *OutboxEventRepo {
	return &OutboxEventRepo{db: db}
}

func (r *OutboxEventRepo) Executor() DBTX {
	return r.db
}

func (r *OutboxEventRepo) Insert(ctx context.Context, event *models.OutboxEvent) (int64, error) {
	const q = `
		INSERT INTO outbox_events (topic, aggregate_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int64
	if err := r.db.QueryRowContext(
		ctx,
		q,
		event.Topic,
		event.AggregateID,
		event.Payload,
		event.OccurredAt,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

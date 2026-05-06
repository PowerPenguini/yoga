package repos

import (
	"context"

	"main/models"
)

type ItemRepo struct {
	db DBTX
}

func NewItemRepo(db DBTX) *ItemRepo {
	return &ItemRepo{db: db}
}

func (r *ItemRepo) Executor() DBTX {
	return r.db
}

func (r *ItemRepo) Insert(ctx context.Context, item models.Item) (int64, error) {
	const q = `INSERT INTO items (name) VALUES ($1) RETURNING id`
	var id int64
	if err := r.db.QueryRowContext(ctx, q, item.Name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *ItemRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM items WHERE name = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, name).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

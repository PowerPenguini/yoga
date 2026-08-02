package repos

import (
	"context"
	"database/sql"

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

func (r *ItemRepo) Insert(ctx context.Context, item *models.Item) (int64, error) {
	const q = `
		INSERT INTO items (name, code, description, archived_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id int64
	if err := r.db.QueryRowContext(
		ctx,
		q,
		item.Name,
		item.Code,
		item.Description,
		item.ArchivedAt,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *ItemRepo) SelectByIDForUpdate(ctx context.Context, id int64) (*models.Item, error) {
	const q = `
		SELECT id, name, code, description, archived_at
		FROM items
		WHERE id = $1
		FOR UPDATE
	`
	var item models.Item
	if err := r.db.QueryRowContext(ctx, q, id).Scan(
		&item.ID,
		&item.Name,
		&item.Code,
		&item.Description,
		&item.ArchivedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepo) ExistsByCodeForUpdate(ctx context.Context, code string) (bool, error) {
	const q = `SELECT id FROM items WHERE code = $1 LIMIT 1 FOR UPDATE`
	var id int64
	if err := r.db.QueryRowContext(ctx, q, code).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ItemRepo) ExistsByCodeOtherForUpdate(ctx context.Context, code string, excludeID int64) (bool, error) {
	const q = `SELECT id FROM items WHERE code = $1 AND id <> $2 LIMIT 1 FOR UPDATE`
	var id int64
	if err := r.db.QueryRowContext(ctx, q, code, excludeID).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ItemRepo) Update(ctx context.Context, item *models.Item) (bool, error) {
	const q = `
		UPDATE items
		SET name = $1, code = $2, description = $3, archived_at = $4
		WHERE id = $5
	`
	result, err := r.db.ExecContext(
		ctx,
		q,
		item.Name,
		item.Code,
		item.Description,
		item.ArchivedAt,
		item.ID,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

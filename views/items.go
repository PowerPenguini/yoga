package views

import (
	"context"
	"database/sql"
	"log"

	"main/contract"
)

type ItemViewer struct {
	db     *sql.DB
	logger *log.Logger
}

func NewItemViewer(db *sql.DB, logger *log.Logger) *ItemViewer {
	return &ItemViewer{db: db, logger: logger}
}

func (v *ItemViewer) List(ctx context.Context) ([]contract.ItemResponse, error) {
	const q = `SELECT id, name FROM items ORDER BY id DESC`
	rows, err := v.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []contract.ItemResponse
	for rows.Next() {
		var item contract.ItemResponse
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

package views

import (
	"context"
	"database/sql"
	"log"

	"github.com/PowerPenguini/errs"

	"main/contract"
)

type ItemListQuery struct {
	IncludeArchived bool
}

type ItemViewer struct {
	db     *sql.DB
	logger *log.Logger
}

func NewItemViewer(db *sql.DB, logger *log.Logger) *ItemViewer {
	return &ItemViewer{db: db, logger: logger}
}

func (v *ItemViewer) List(ctx context.Context, query ItemListQuery) ([]contract.ItemResponse, error) {
	const q = `
		SELECT id, name, code, description, archived_at
		FROM items
		WHERE $1 OR archived_at IS NULL
		ORDER BY id DESC
	`
	rows, err := v.db.QueryContext(ctx, q, query.IncludeArchived)
	if err != nil {
		v.logger.Printf("list items query failed: %v", err)
		return nil, errs.NewError("item_list_failed", "failed to list items", errs.InternalType, err)
	}
	defer rows.Close()

	out := make([]contract.ItemResponse, 0)
	for rows.Next() {
		var item contract.ItemResponse
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Code,
			&item.Description,
			&item.ArchivedAt,
		); err != nil {
			v.logger.Printf("scan item failed: %v", err)
			return nil, errs.NewError("item_list_failed", "failed to list items", errs.InternalType, err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		v.logger.Printf("iterate items failed: %v", err)
		return nil, errs.NewError("item_list_failed", "failed to list items", errs.InternalType, err)
	}
	return out, nil
}

package logic

import (
	"context"
	"strings"

	"main/di"
	"main/models"
)

type CreateItem struct {
	Name string
}

type CreateItemResult struct {
	Item models.Item
}

func (c CreateItem) Execute(ctx context.Context, deps *di.DI) (*CreateItemResult, error) {
	item := models.Item{Name: strings.TrimSpace(c.Name)}

	result, err := di.ExecuteInTx(deps, func(txDI *di.DI) (*CreateItemResult, error) {
		if err := txDI.ItemValidator.Validate(ctx, item); err != nil {
			return nil, err
		}

		id, err := txDI.ItemRepo.Insert(ctx, item)
		if err != nil {
			return nil, err
		}
		item.ID = id
		return &CreateItemResult{Item: item}, nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

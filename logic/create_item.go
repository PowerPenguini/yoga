package logic

import (
	"context"
	"errors"
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

func (c CreateItem) Normalize() CreateItem {
	c.Name = strings.TrimSpace(c.Name)
	return c
}

func (c CreateItem) Validate(ctx context.Context, deps *di.DI) error {
	exists, err := deps.ItemRepo.ExistsByName(ctx, c.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("item name already exists")
	}
	return nil
}

func (c CreateItem) Execute(ctx context.Context, deps *di.DI) (*CreateItemResult, error) {
	c = c.Normalize()
	item := models.Item{Name: c.Name}

	result, err := di.ExecuteInTx(deps, func(txDI *di.DI) (*CreateItemResult, error) {
		if err := txDI.ItemValidator.Validate(item); err != nil {
			return nil, err
		}
		if err := c.Validate(ctx, txDI); err != nil {
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

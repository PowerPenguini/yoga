package logic

import (
	"context"
	"strings"

	"github.com/PowerPenguini/errs"

	"main/di"
	"main/models"
)

type CreateItem struct {
	Name        string
	Code        string
	Description *string
}

type CreateItemResult struct {
	ID          int64
	Name        string
	Code        string
	Description *string
}

func (c *CreateItem) Normalize() {
	c.Name = strings.TrimSpace(c.Name)
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	c.Description = normalizeOptionalText(c.Description)
}

func (c *CreateItem) Validate(ctx context.Context, deps *di.DI) error {
	exists, err := deps.ItemRepo.ExistsByCodeForUpdate(ctx, c.Code)
	if err != nil {
		return errs.NewError("item_code_check_failed", "failed to verify item code", errs.InternalType, err)
	}
	if exists {
		return errs.NewFieldError("item_code_already_exists", "code", "item code already exists", errs.ValidationType, nil)
	}
	return nil
}

func (c *CreateItem) Execute(ctx context.Context, deps *di.DI) (*CreateItemResult, error) {
	return di.ExecuteInTx(deps, func(txDI *di.DI) (*CreateItemResult, error) {
		c.Normalize()
		item := &models.Item{
			Name:        c.Name,
			Code:        c.Code,
			Description: c.Description,
		}

		if err := txDI.ItemValidator.Validate(item); err != nil {
			return nil, err
		}
		if err := c.Validate(ctx, txDI); err != nil {
			return nil, err
		}

		id, err := txDI.ItemRepo.Insert(ctx, item)
		if err != nil {
			return nil, errs.NewError("item_create_failed", "failed to create item", errs.InternalType, err)
		}
		item.ID = id

		event := EnqueueItemEvent{
			Topic:  ItemCreatedTopic,
			ItemID: item.ID,
			Data: map[string]any{
				"code": item.Code,
				"name": item.Name,
			},
		}
		if err := event.Execute(ctx, txDI); err != nil {
			return nil, err
		}

		return &CreateItemResult{
			ID:          item.ID,
			Name:        item.Name,
			Code:        item.Code,
			Description: item.Description,
		}, nil
	})
}

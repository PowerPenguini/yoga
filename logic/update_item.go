package logic

import (
	"context"
	"strings"

	"github.com/PowerPenguini/errs"

	"main/di"
	"main/models"
)

type UpdateItem struct {
	ID          int64
	Name        *string
	Code        *string
	Description *string
}

type UpdateItemResult struct {
	ID          int64
	Name        string
	Code        string
	Description *string
}

func (u *UpdateItem) Normalize() {
	u.Name = normalizeRequiredUpdateText(u.Name)
	u.Code = normalizeCodeUpdate(u.Code)
	u.Description = normalizeOptionalUpdateText(u.Description)
}

func (u *UpdateItem) Validate(ctx context.Context, deps *di.DI) (*models.Item, error) {
	if u.ID <= 0 {
		return nil, errs.NewFieldError("item_id_required", "id", "item id is required", errs.ValidationType, nil)
	}
	if u.Name == nil && u.Code == nil && u.Description == nil {
		return nil, errs.NewError("item_update_empty", "at least one field is required", errs.ValidationType, nil)
	}

	existing, err := deps.ItemRepo.SelectByIDForUpdate(ctx, u.ID)
	if err != nil {
		return nil, errs.NewError("item_lookup_failed", "failed to load item", errs.InternalType, err)
	}
	if existing == nil {
		return nil, errs.NewError("item_not_found", "item not found", errs.NotFoundType, nil)
	}
	if existing.ArchivedAt != nil {
		return nil, errs.NewError("item_archived", "archived item cannot be updated", errs.ValidationType, nil)
	}

	if u.Code != nil {
		exists, err := deps.ItemRepo.ExistsByCodeOtherForUpdate(ctx, *u.Code, u.ID)
		if err != nil {
			return nil, errs.NewError("item_code_check_failed", "failed to verify item code", errs.InternalType, err)
		}
		if exists {
			return nil, errs.NewFieldError("item_code_already_exists", "code", "item code already exists", errs.ValidationType, nil)
		}
	}
	return existing, nil
}

func (u *UpdateItem) Execute(ctx context.Context, deps *di.DI) (*UpdateItemResult, error) {
	return di.ExecuteInTx(deps, func(txDI *di.DI) (*UpdateItemResult, error) {
		u.Normalize()
		item, err := u.Validate(ctx, txDI)
		if err != nil {
			return nil, err
		}

		if u.Name != nil {
			item.Name = *u.Name
		}
		if u.Code != nil {
			item.Code = *u.Code
		}
		if u.Description != nil {
			item.Description = optionalTextValue(*u.Description)
		}

		if err := txDI.ItemValidator.Validate(item); err != nil {
			return nil, err
		}
		updated, err := txDI.ItemRepo.Update(ctx, item)
		if err != nil {
			return nil, errs.NewError("item_update_failed", "failed to update item", errs.InternalType, err)
		}
		if !updated {
			return nil, errs.NewError("item_not_found", "item not found", errs.NotFoundType, nil)
		}

		event := EnqueueItemEvent{
			Topic:  ItemUpdatedTopic,
			ItemID: item.ID,
			Data: map[string]any{
				"code": item.Code,
				"name": item.Name,
			},
		}
		if err := event.Execute(ctx, txDI); err != nil {
			return nil, err
		}

		return &UpdateItemResult{
			ID:          item.ID,
			Name:        item.Name,
			Code:        item.Code,
			Description: item.Description,
		}, nil
	})
}

func normalizeRequiredUpdateText(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	return &normalized
}

func normalizeCodeUpdate(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(*value))
	return &normalized
}

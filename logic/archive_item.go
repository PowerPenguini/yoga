package logic

import (
	"context"

	"github.com/PowerPenguini/errs"

	"main/di"
	"main/models"
)

type ArchiveItem struct {
	ID int64
}

func (a *ArchiveItem) Validate(ctx context.Context, deps *di.DI) (*models.Item, error) {
	if a.ID <= 0 {
		return nil, errs.NewFieldError("item_id_required", "id", "item id is required", errs.ValidationType, nil)
	}
	item, err := deps.ItemRepo.SelectByIDForUpdate(ctx, a.ID)
	if err != nil {
		return nil, errs.NewError("item_lookup_failed", "failed to load item", errs.InternalType, err)
	}
	if item == nil {
		return nil, errs.NewError("item_not_found", "item not found", errs.NotFoundType, nil)
	}
	if item.ArchivedAt != nil {
		return nil, errs.NewError("item_already_archived", "item is already archived", errs.ValidationType, nil)
	}
	return item, nil
}

func (a *ArchiveItem) Execute(ctx context.Context, deps *di.DI) error {
	return di.ExecuteInTxNoResult(deps, func(txDI *di.DI) error {
		item, err := a.Validate(ctx, txDI)
		if err != nil {
			return err
		}

		now := txDI.Clock.Now()
		item.ArchivedAt = &now
		if err := txDI.ItemValidator.Validate(item); err != nil {
			return err
		}
		updated, err := txDI.ItemRepo.Update(ctx, item)
		if err != nil {
			return errs.NewError("item_archive_failed", "failed to archive item", errs.InternalType, err)
		}
		if !updated {
			return errs.NewError("item_not_found", "item not found", errs.NotFoundType, nil)
		}

		event := EnqueueItemEvent{
			Topic:  ItemArchivedTopic,
			ItemID: item.ID,
			Data:   map[string]any{"code": item.Code},
		}
		return event.Execute(ctx, txDI)
	})
}

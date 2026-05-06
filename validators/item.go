package validators

import (
	"context"
	"errors"
	"strings"

	"main/models"
	"main/repos"
)

type ItemLookup interface {
	ExistsByName(context.Context, string) (bool, error)
}

type ItemValidator struct {
	repo ItemLookup
}

func NewItemValidator(repo ItemLookup) *ItemValidator {
	return &ItemValidator{repo: repo}
}

func (v *ItemValidator) Repo() ItemLookup {
	return v.repo
}

func (v *ItemValidator) Validate(ctx context.Context, item models.Item) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("name is required")
	}

	exists, err := v.repo.ExistsByName(ctx, item.Name)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("item name already exists")
	}
	return nil
}

var _ ItemLookup = (*repos.ItemRepo)(nil)

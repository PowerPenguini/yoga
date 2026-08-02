package validators

import (
	"errors"
	"strings"

	"main/models"
)

type ItemValidator struct{}

func NewItemValidator() *ItemValidator {
	return &ItemValidator{}
}

func (v *ItemValidator) Validate(item models.Item) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

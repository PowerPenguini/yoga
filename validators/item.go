package validators

import (
	"strings"

	"github.com/PowerPenguini/errs"

	"main/models"
)

const (
	maxItemNameLength        = 120
	maxItemCodeLength        = 32
	maxItemDescriptionLength = 1000
)

type ItemValidator struct{}

func NewItemValidator() *ItemValidator {
	return &ItemValidator{}
}

func (v *ItemValidator) Validate(item *models.Item) error {
	return v.ValidateFields([]string{"name", "code", "description"}, item)
}

func (v *ItemValidator) ValidateFields(fields []string, item *models.Item) error {
	if item == nil {
		return errs.NewError("item_required", "item is required", errs.ValidationType, nil)
	}

	selected := make(map[string]bool, len(fields))
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		switch field {
		case "name", "code", "description":
			selected[field] = true
		default:
			return errs.NewFieldError("item_field_invalid", "fields", "unsupported item field", errs.ValidationType, nil)
		}
	}
	if len(selected) == 0 {
		return errs.NewError("item_fields_required", "fields are required", errs.ValidationType, nil)
	}

	validationErrors := errs.NewErrorList()
	if selected["name"] {
		name := strings.TrimSpace(item.Name)
		switch {
		case name == "":
			validationErrors.Append(errs.NewFieldError("item_name_required", "name", "name is required", errs.ValidationType, nil))
		case len(name) > maxItemNameLength:
			validationErrors.Append(errs.NewFieldError("item_name_too_long", "name", "name is too long", errs.ValidationType, nil))
		}
	}

	if selected["code"] {
		code := strings.TrimSpace(item.Code)
		switch {
		case code == "":
			validationErrors.Append(errs.NewFieldError("item_code_required", "code", "code is required", errs.ValidationType, nil))
		case len(code) > maxItemCodeLength:
			validationErrors.Append(errs.NewFieldError("item_code_too_long", "code", "code is too long", errs.ValidationType, nil))
		case !isItemCode(code):
			validationErrors.Append(errs.NewFieldError("item_code_invalid", "code", "code may contain only uppercase letters, digits, and hyphens", errs.ValidationType, nil))
		}
	}

	if selected["description"] && item.Description != nil && len(*item.Description) > maxItemDescriptionLength {
		validationErrors.Append(errs.NewFieldError("item_description_too_long", "description", "description is too long", errs.ValidationType, nil))
	}

	if validationErrors.Len() > 0 {
		return validationErrors
	}
	return nil
}

func isItemCode(code string) bool {
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

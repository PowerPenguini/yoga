package validators

import (
	"errors"
	"testing"

	"github.com/PowerPenguini/errs"

	"main/models"
)

func TestItemValidatorValidateAcceptsValidEntity(t *testing.T) {
	description := "example"
	item := &models.Item{
		Name:        "Example item",
		Code:        "ITEM-01",
		Description: &description,
	}

	if err := NewItemValidator().Validate(item); err != nil {
		t.Fatalf("valid item: %v", err)
	}
}

func TestItemValidatorValidateCollectsEntityErrors(t *testing.T) {
	item := &models.Item{Name: "  ", Code: "invalid code"}

	err := NewItemValidator().Validate(item)
	var list *errs.ErrorList
	if !errors.As(err, &list) {
		t.Fatalf("error = %T %v, want *errs.ErrorList", err, err)
	}
	if list.Len() != 2 {
		t.Fatalf("validation errors = %d, want 2", list.Len())
	}
}

func TestItemValidatorValidateFieldsChecksOnlySelectedFields(t *testing.T) {
	item := &models.Item{Name: "", Code: "ITEM-01"}

	if err := NewItemValidator().ValidateFields([]string{"code"}, item); err != nil {
		t.Fatalf("selected code field: %v", err)
	}
}

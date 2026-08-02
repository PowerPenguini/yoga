package validators

import (
	"testing"

	"main/models"
)

func TestItemValidatorValidatesEntityRules(t *testing.T) {
	validator := NewItemValidator()

	if err := validator.Validate(models.Item{Name: "example"}); err != nil {
		t.Fatalf("valid item: %v", err)
	}
	if err := validator.Validate(models.Item{Name: "   "}); err == nil {
		t.Fatal("blank item name passed validation")
	}
}

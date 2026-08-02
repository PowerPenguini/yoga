package logic

import "testing"

func TestCreateItemNormalize(t *testing.T) {
	description := "  example description  "
	action := CreateItem{
		Name:        "  Example item  ",
		Code:        " item-01 ",
		Description: &description,
	}

	action.Normalize()

	if action.Name != "Example item" {
		t.Fatalf("normalized name = %q, want %q", action.Name, "Example item")
	}
	if action.Code != "ITEM-01" {
		t.Fatalf("normalized code = %q, want %q", action.Code, "ITEM-01")
	}
	if action.Description == nil || *action.Description != "example description" {
		t.Fatalf("normalized description = %v, want %q", action.Description, "example description")
	}
}

func TestCreateItemNormalizeClearsBlankOptionalDescription(t *testing.T) {
	description := "   "
	action := CreateItem{Description: &description}

	action.Normalize()

	if action.Description != nil {
		t.Fatalf("normalized description = %q, want nil", *action.Description)
	}
}

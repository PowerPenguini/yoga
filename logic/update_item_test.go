package logic

import "testing"

func TestUpdateItemNormalizePreservesPatchIntent(t *testing.T) {
	name := "  New name  "
	code := " new-code "
	description := "   "
	action := UpdateItem{
		Name:        &name,
		Code:        &code,
		Description: &description,
	}

	action.Normalize()

	if action.Name == nil || *action.Name != "New name" {
		t.Fatalf("normalized name = %v, want %q", action.Name, "New name")
	}
	if action.Code == nil || *action.Code != "NEW-CODE" {
		t.Fatalf("normalized code = %v, want %q", action.Code, "NEW-CODE")
	}
	if action.Description == nil || *action.Description != "" {
		t.Fatalf("normalized description = %v, want pointer to empty string", action.Description)
	}
}

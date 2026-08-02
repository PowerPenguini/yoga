package logic

import "testing"

func TestCreateItemNormalizeTrimsName(t *testing.T) {
	action := CreateItem{Name: "  example  "}.Normalize()
	if action.Name != "example" {
		t.Fatalf("normalized name = %q, want %q", action.Name, "example")
	}
}

package di

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type guardrailRepoGroup struct {
	ItemRepo *int
	Other    *int
	hidden   *int
	Count    int
}

func TestMissingNilPtrFieldsReportsInvalidWiring(t *testing.T) {
	value := 1

	problems := missingNilPtrFields(reflectValue(guardrailRepoGroup{
		ItemRepo: &value,
		Count:    2,
	}), "repo")

	msg := strings.Join(problems, ", ")
	for _, want := range []string{
		"repo.Other",
		"repo.Count (expected pointer, got int)",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("problems %q do not contain %q", msg, want)
		}
	}
	if strings.Contains(msg, "hidden") {
		t.Fatalf("problems %q include unexported field", msg)
	}
}

func TestMustValidateWiringPanicsWhenDIIsIncomplete(t *testing.T) {
	deps := &DI{}

	msg := mustPanicMessage(t, deps.MustValidateWiring)
	if !strings.Contains(msg, "invalid DI wiring:") {
		t.Fatalf("panic %q does not contain wiring prefix", msg)
	}
	if !strings.Contains(msg, "repo.ItemRepo") || !strings.Contains(msg, "val.ItemValidator") {
		t.Fatalf("panic %q does not include missing repo and validator", msg)
	}
}

func reflectValue(v any) reflect.Value {
	return reflect.ValueOf(v)
}

func mustPanicMessage(t *testing.T, fn func()) (message string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			message = fmt.Sprint(r)
			return
		}
		t.Fatal("expected panic")
	}()
	fn()
	return ""
}

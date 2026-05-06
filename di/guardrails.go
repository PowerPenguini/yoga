package di

import (
	"fmt"
	"reflect"
	"strings"
)

func (d *DI) MustValidateWiring() {
	var problems []string
	problems = append(problems, missingNilPtrFields(reflect.ValueOf(d.repo), "repo")...)
	problems = append(problems, missingNilPtrFields(reflect.ValueOf(d.val), "val")...)
	if len(problems) > 0 {
		panic(fmt.Sprintf("invalid DI wiring: %s", strings.Join(problems, ", ")))
	}
}

func missingNilPtrFields(v reflect.Value, prefix string) []string {
	if !v.IsValid() {
		return []string{prefix + " (nil)"}
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return []string{prefix + " (nil)"}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []string{prefix + " (not a struct)"}
	}

	t := v.Type()
	var out []string
	for i := 0; i < v.NumField(); i++ {
		ft := t.Field(i)
		if ft.PkgPath != "" {
			continue
		}
		fv := v.Field(i)

		if fv.Kind() != reflect.Pointer {
			out = append(out, fmt.Sprintf("%s.%s (expected pointer, got %s)", prefix, ft.Name, fv.Kind()))
			continue
		}
		if fv.IsNil() {
			out = append(out, fmt.Sprintf("%s.%s", prefix, ft.Name))
		}
	}
	return out
}

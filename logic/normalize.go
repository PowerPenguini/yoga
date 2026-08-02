package logic

import "strings"

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	return optionalTextValue(strings.TrimSpace(*value))
}

func normalizeOptionalUpdateText(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	return &normalized
}

func optionalTextValue(value string) *string {
	if value == "" {
		return nil
	}
	normalized := value
	return &normalized
}

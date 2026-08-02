package models

import "time"

type Item struct {
	ID          int64
	Name        string
	Code        string
	Description *string
	ArchivedAt  *time.Time
}

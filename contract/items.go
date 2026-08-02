package contract

import "time"

type CreateItemRequest struct {
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description,omitempty"`
}

type UpdateItemRequest struct {
	Name        *string `json:"name,omitempty"`
	Code        *string `json:"code,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ItemResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description *string    `json:"description,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

type CreateItemResponse struct {
	Item ItemResponse `json:"item"`
}

type UpdateItemResponse struct {
	Item ItemResponse `json:"item"`
}

type ListItemsResponse struct {
	Items []ItemResponse `json:"items"`
}

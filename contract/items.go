package contract

type CreateItemRequest struct {
	Name string `json:"name"`
}

type ItemResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CreateItemResponse struct {
	Item ItemResponse `json:"item"`
}

type ListItemsResponse struct {
	Items []ItemResponse `json:"items"`
}

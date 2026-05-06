package handlers

import (
	"encoding/json"
	"net/http"

	"main/contract"
	"main/di"
	"main/logic"
)

type ItemsHandler struct {
	deps *di.DI
}

func NewItemsHandler(deps *di.DI) *ItemsHandler {
	return &ItemsHandler{deps: deps}
}

func (h *ItemsHandler) PostItem(w http.ResponseWriter, r *http.Request) {
	var req contract.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := logic.CreateItem{Name: req.Name}.Execute(r.Context(), h.deps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, contract.CreateItemResponse{
		Item: contract.ItemResponse{ID: result.Item.ID, Name: result.Item.Name},
	})
}

func (h *ItemsHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.deps.ItemViewer.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list items", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, contract.ListItemsResponse{Items: items})
}

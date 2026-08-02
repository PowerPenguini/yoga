package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/PowerPenguini/errs"

	"main/contract"
	"main/di"
	"main/logic"
	"main/views"
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
		errs.WriteError(w, errs.NewError("request_body_invalid", "invalid request body", errs.BadRequestType, err))
		return
	}

	action := logic.CreateItem{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}
	result, err := action.Execute(r.Context(), h.deps)
	if err != nil {
		errs.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, contract.CreateItemResponse{
		Item: contract.ItemResponse{
			ID:          result.ID,
			Name:        result.Name,
			Code:        result.Code,
			Description: result.Description,
		},
	})
}

func (h *ItemsHandler) PatchItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseItemID(w, r)
	if !ok {
		return
	}

	var req contract.UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errs.WriteError(w, errs.NewError("request_body_invalid", "invalid request body", errs.BadRequestType, err))
		return
	}

	action := logic.UpdateItem{
		ID:          id,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}
	result, err := action.Execute(r.Context(), h.deps)
	if err != nil {
		errs.WriteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, contract.UpdateItemResponse{
		Item: contract.ItemResponse{
			ID:          result.ID,
			Name:        result.Name,
			Code:        result.Code,
			Description: result.Description,
		},
	})
}

func (h *ItemsHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseItemID(w, r)
	if !ok {
		return
	}
	action := logic.ArchiveItem{ID: id}
	if err := action.Execute(r.Context(), h.deps); err != nil {
		errs.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ItemsHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	includeArchived := false
	if raw := strings.TrimSpace(r.URL.Query().Get("include_archived")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			errs.WriteError(w, errs.NewFieldError("include_archived_invalid", "include_archived", "include_archived must be a boolean", errs.BadRequestType, err))
			return
		}
		includeArchived = parsed
	}

	items, err := h.deps.ItemViewer.List(r.Context(), views.ItemListQuery{
		IncludeArchived: includeArchived,
	})
	if err != nil {
		errs.WriteError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contract.ListItemsResponse{Items: items})
}

func parseItemID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("item_id"), 10, 64)
	if err != nil || id <= 0 {
		errs.WriteError(w, errs.NewFieldError("item_id_invalid", "item_id", "item id must be a positive integer", errs.BadRequestType, err))
		return 0, false
	}
	return id, true
}

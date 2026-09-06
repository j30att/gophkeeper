// Package handlers содержит generated HTTP-handlers infra API.
package handlers

import (
	"context"

	"github.com/igor/gophkeeper/pkg/api/generated/infra"
)

// Handler реализует generated infra.StrictServerInterface.
type Handler struct{}

// New создает Handler.
func New() *Handler {
	return &Handler{}
}

// Get возвращает health check.
func (h *Handler) Get(_ context.Context, _ infra.GetRequestObject) (infra.GetResponseObject, error) {
	return infra.Get200JSONResponse{Status: "ok"}, nil
}

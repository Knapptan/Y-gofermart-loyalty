package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// Create — POST /api/user/orders
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	number := string(body)

	err = h.svc.CreateOrder(r.Context(), userID, number)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted) // 202
	case errors.Is(err, service.ErrInvalidOrderNumber):
		writeError(w, http.StatusUnprocessableEntity, err.Error()) // 422
	case errors.Is(err, repository.ErrOrderExistsSame):
		w.WriteHeader(http.StatusOK) // 200
	case errors.Is(err, repository.ErrOrderExistsOther):
		writeError(w, http.StatusConflict, err.Error()) // 409
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// List — GET /api/user/orders
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orders, err := h.svc.ListOrders(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

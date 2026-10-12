package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

type BalanceHandler struct {
	svc *service.BalanceService
}

func NewBalanceHandler(svc *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

// Get — GET /api/user/balance
func (h *BalanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	b, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// Withdraw — POST /api/user/balance/withdraw
func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.svc.Withdraw(r.Context(), userID, req.Order, req.Sum)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrInvalidOrderNumber):
		writeError(w, http.StatusUnprocessableEntity, err.Error()) // 422
	case errors.Is(err, service.ErrInvalidWithdrawSum):
		writeError(w, http.StatusBadRequest, err.Error()) // 400
	case errors.Is(err, service.ErrInsufficientFunds):
		writeError(w, http.StatusPaymentRequired, err.Error()) // 402
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// ListWithdrawals — GET /api/user/withdrawals
func (h *BalanceHandler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := h.svc.ListWithdrawals(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}
	writeJSON(w, http.StatusOK, list)
}

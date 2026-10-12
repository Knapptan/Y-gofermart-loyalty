package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

// UserHandler — HTTP-хендлер регистрации и логина.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler создаёт UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type credentialsRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register — POST /api/user/register
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.svc.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyCredentials):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrLoginExists):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

// Login — POST /api/user/login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyCredentials):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

package user

import (
	"net/http"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
	"github.com/gorilla/mux"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/login", h.handleLogin).Methods("POST")
	router.HandleFunc("/register", h.handleRegister).Methods("POST")
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Get JSON paylaod
	var payload types.RegisterUserPayload

	// Show an error if it exists
	if err := utils.ParseJSON(r, payload); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
	}

	// Check if the user exists
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
}

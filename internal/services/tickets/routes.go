package tickets

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diegob0/rspv_backend/internal/services/auth"
	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
	"github.com/gorilla/mux"
)

type Handler struct {
	store types.TicketStore
}

func NewHandler(store types.TicketStore) *Handler {
	return &Handler{store: store}
}

// Router handler
func (h *Handler) RegisterRoutes(router *mux.Router) {
	protected := router.PathPrefix("/tickets").Subrouter()
	protected.Use(auth.AuthMiddleware)

	// Public routes
	// Get the qr code and metadata
	router.HandleFunc("/tickets/info/{name}", h.handleGetGuestData).Methods(http.MethodGet)

	// TODO: Get info about the count of named, general and the sum of all tickets
	// protected.HandleFunc("/count/info", h.handleGetCountNamed).Methods(http.MethodGet)

	// TODO: Create an activate all endpoint
	// protected.HandleFunc("/activate/all", h.handleGenerateNamedTickets).Methods(http.MethodGet)
	protected.HandleFunc("/regenerate/{id}", h.handleRegenerateTicket).Methods(http.MethodGet)
	protected.HandleFunc("/activate/{id}", h.handleActivateTickets).Methods(http.MethodGet)
	// protected.HandleFunc("/generate-generals/{id}", h.handleAcivateGenerals).Methods(http.MethodGet)

	// TODO: Get data about the tickets
	// protected.HandleFunc("/generals", h.handleGenerateNamedTickets).Methods(http.MethodGet)
	// protected.HandleFunc("/named", h.handleGenerateNamedTickets).Methods(http.MethodGet)
}

// @Summary Return the guest metadata
// @Description Return the guest tickets
// @Tags tickets
// @Param name path string true "Guest Name"
// @Param confirmAttendance query bool false "Confirm attendance (true/false)"
// @Param email query string false "Optional email to send the ticket PDF"
// @Success 200 {array} types.ReturnGuestMetadata
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/info/{name} [get]
func (h *Handler) handleGetGuestData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guestName := vars["name"]
	if guestName == "" {

		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("guest name is required"))
		return
	}

	// Get the query parameter confirmAttendance as string
	confirmStr := r.URL.Query().Get("confirmAttendance")

	// Parse string to bool, default to false if empty or invalid
	confirmAttendance := false
	var err error
	if confirmStr != "" {
		confirmAttendance, err = strconv.ParseBool(confirmStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid confirmAttendance value"))
			return
		}
	}

	// Get optional email query param
	email := r.URL.Query().Get("email")

	// Call GenerateTickets using guestName instead of ID
	t, err := h.store.GetTicketInfo(guestName, confirmAttendance, email)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return

	}

	utils.WriteJSON(w, http.StatusOK, t)
}

// @Summary Generate the tickets per guest by ID
// @Description Generate the tickets and stores the urls into the guest table
// @Tags tickets
// @Security BearerAuth
// @Param id path int true "Guest ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/activate/{id} [get]
func (h *Handler) handleActivateTickets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid user ID"))

		return
	}

	err = h.store.GenerateTickets(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) handleRegenerateTicket(w http.ResponseWriter, r *http.Request) {
	// TODO: Things to do
}

// @Summary Get the count for tickets
// @Description Returns the count for tickets
// @Tags tickets
// @Security BearerAuth
// @Produce json
// @Success 200 {array} types.Tcikets
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/all [get]
// func (h *Handler) handleCountAllTickets(w http.ResponseWriter, r *http.Request) {
// 	u, err := h.store.GetUsers()
// 	if err != nil {
// 		utils.WriteError(w, http.StatusInternalServerError, err)
// 		return
// 	}
//
// 	utils.WriteJSON(w, http.StatusOK, u)
// }

// @Summary Get the count for tickets that are related to guests
// @Description Returns the count for tickets related to guests
// @Tags tickets
// @Security BearerAuth
// @Produce json
// @Success 200 {array} types.Tcikets
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/named [get]
// func (h *Handler) handleCountNamedTickets(w http.ResponseWriter, r *http.Request) {
// 	u, err := h.store.GetUsers()
// 	if err != nil {
// 		utils.WriteError(w, http.StatusInternalServerError, err)
// 		return
// 	}
//
// 	utils.WriteJSON(w, http.StatusOK, u)
// }

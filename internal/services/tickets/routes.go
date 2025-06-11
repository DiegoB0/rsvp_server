package tickets

import (
	"fmt"
	"net/http"
	"strconv"

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
	// Generate tickets for guests
	router.HandleFunc("/tickets/generate/{id}", h.handleGenerateNamedTickets).Methods(http.MethodGet)

	// TODO: Generate general tickets(Not related to guests), Get Count of tickets. One for both types. One to get
	// by named and one to get generals
}

// @Summary Generate the tickets per guest by ID
// @Description Returns the tickets in PDF format for the given guest ID
// @Tags tickets
// @Security BearerAuth
// @Produce application/pdf
// @Param id path int true "Guest ID"
// @Param confirmAttendance query bool false "Confirm attendance (true/false)"
// @Success 200 {file} file
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/generate/{id} [get]
func (h *Handler) handleGenerateNamedTickets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guest ID"))
		return
	}

	// Get the query parameter confirmAttendance as string
	confirmStr := r.URL.Query().Get("confirmAttendance")

	// Parse string to bool, default to false if empty or invalid
	confirmAttendance := false
	if confirmStr != "" {
		confirmAttendance, err = strconv.ParseBool(confirmStr)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid confirmAttendance value"))
			return

		}
	}
	pdfBytes, err := h.store.GenerateTickets(id, confirmAttendance)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Proper headers for PDF response
	w.Header().Set("Content-Type", "application/pdf")
	// w.Header().Set("Content-Disposition", "inline; filename=\"tickets.pdf\"")
	w.Header().Set("Content-Disposition", "attachment; filename=\"tickets.pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
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

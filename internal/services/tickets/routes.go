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

	// Get the url of the pdf file
	// router.HandleFunc("/tickets/download/{name}", h.handleGenerateNamedTickets).Methods(http.MethodGet)

	protected.HandleFunc("/activate/{id}", h.handleActivateTickets).Methods(http.MethodGet)
	// protected.HandleFunc("/activate-all", h.handleGenerateNamedTickets).Methods(http.MethodGet)
	//
	// protected.HandleFunc("/regenerate/{id}", h.handleGenerateNamedTickets).Methods(http.MethodGet)
	// protected.HandleFunc("/generals", h.handleGenerateNamedTickets).Methods(http.MethodGet)

	// TODO: Generate general tickets(Not related to guests), Get Count of tickets. One for both types. One to get
	// by named and one to get generals
}

// @Summary Generate the tickets per guest by name
// @Description Returns the tickets in PDF format for the given guest name
// @Tags tickets
// @Produce application/pdf
// @Param name path string true "Guest name"
// @Param confirmAttendance query bool false "Confirm attendance (true/false)"
// @Success 200 {file} file
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/generate/{name} [get]
// func (h *Handler) handleGenerateNamedTickets(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	guestName := vars["name"]
// 	if guestName == "" {
//
// 		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("guest name is required"))
// 		return
// 	}
//
// 	// Get the query parameter confirmAttendance as string
// 	confirmStr := r.URL.Query().Get("confirmAttendance")
//
// 	// Parse string to bool, default to false if empty or invalid
// 	confirmAttendance := false
// 	var err error
// 	if confirmStr != "" {
// 		confirmAttendance, err = strconv.ParseBool(confirmStr)
// 		if err != nil {
// 			utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid confirmAttendance value"))
// 			return
// 		}
// 	}
//
// 	// Call GenerateTickets using guestName instead of ID
// 	pdfBytes, err := h.store.GenerateTickets(guestName, confirmAttendance)
// 	if err != nil {
//
// 		utils.WriteError(w, http.StatusInternalServerError, err)
// 		return
//
// 	}
//
// 	// Proper headers for PDF response
// 	w.Header().Set("Content-Type", "application/pdf")
// 	w.Header().Set("Content-Disposition", "attachment; filename=\"tickets.pdf\"")
// 	w.WriteHeader(http.StatusOK)
// 	_, _ = w.Write(pdfBytes)
// }

// @Summary Return the guest metadata
// @Description Return the guest tickets
// @Tags tickets
// @Param name path string true "Guest Name"
// @Param confirmAttendance query bool false "Confirm attendance (true/false)"
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

	// Call GenerateTickets using guestName instead of ID
	t, err := h.store.GetTicketInfo(guestName, confirmAttendance)
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

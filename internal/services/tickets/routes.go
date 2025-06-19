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
	router.HandleFunc("/tickets/info/{name}", h.handleGetGuestData).Methods(http.MethodGet)

	protected.HandleFunc("/regenerate/{id}", h.handleRegenerateTicket).Methods(http.MethodGet)
	protected.HandleFunc("/activate/{id}", h.handleActivateTickets).Methods(http.MethodGet)
	protected.HandleFunc("/scan-qr/{code}", h.handleScanTicket).Methods(http.MethodGet)

	// TODO: Get info about the count of named, general and the sum of all tickets
	// protected.HandleFunc("/count/info", h.handleGetCountNamed).Methods(http.MethodGet)

	protected.HandleFunc("/activate-all", h.handleActivateAll).Methods(http.MethodGet)

	// TODO: Logic to handle generals. Create one by one (or a bunch by one operation)
	// protected.HandleFunc("/create-generals", h.handleActivateGenerals).Methods(http.MethodPost)
	// protected.HandleFunc("/generate-generals", h.handleAcivateGenerals).Methods(http.MethodsPost)

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

	err = h.store.GenerateTicket(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Summary Generate all the tickets
// @Description Generate all the tickets that have not being generated yet
// @Tags tickets
// @Security BearerAuth
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/activate-all [get]
func (h *Handler) handleActivateAll(w http.ResponseWriter, r *http.Request) {
	err := h.store.GenerateAllTickets()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Summary Regenerate a ticket per guest by ID
// @Description Regenerate a ticket that has been already been generated
// @Tags tickets
// @Security BearerAuth
// @Param id path int true "Guest ID"
// @Success 200 {file} file "PDF Ticket"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/regenerate/{id} [get]
func (h *Handler) handleRegenerateTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid user ID"))
		return
	}

	pdfData, err := h.store.RegenerateTicket(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"ticket.pdf\"")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)
}

// @Summary Scan a ticket by QR code
// @Description Validates a ticket code, marks it as used, and returns guest and table info.
// @Tags tickets
// @Security BearerAuth
// @Param code path string true "Ticket Code"
// @Success 200 {object} types.ReturnScanedData
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 409 {object} types.ErrorResponse // Already used
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/scan-qr/{code} [get]
func (h *Handler) handleScanTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	result, err := h.store.ScanQR(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

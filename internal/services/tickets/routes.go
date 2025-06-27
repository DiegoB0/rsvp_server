package tickets

import (
	"fmt"
	"log"
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

	protected.HandleFunc("/generate-general/{id}", h.handleGenerateGenerals).Methods(http.MethodGet)
	protected.HandleFunc("/create-generals", h.handleActivateGenerals).Methods(http.MethodPost)

	// Activate all guest tickets in the same operation
	protected.HandleFunc("/activate-all", h.handleActivateAll).Methods(http.MethodGet)

	protected.HandleFunc("/generals", h.handleGetGeneralsInfo).Methods(http.MethodGet)
	// protected.HandleFunc("/named", h.handleGetNamedInfo).Methods(http.MethodGet)

	protected.HandleFunc("/count", h.handleGetTicketsCount).Methods(http.MethodGet)
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
// @Success 200 {object} types.ReturnScannedData
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

// @Summary Create general tickets
// @Description Generates general tickets and enqueues background jobs for QR and PDF upload.
// @Tags tickets
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param count query int true "Number of general tickets to generate"
// @Success 200 {object} map[string]string
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/create-generals [post]/
func (h *Handler) handleActivateGenerals(w http.ResponseWriter, r *http.Request) {
	countStr := r.URL.Query().Get("count")
	if countStr == "" {
		http.Error(w, "Missing 'count' query param", http.StatusBadRequest)
		return
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		http.Error(w, "'count' must be a positive integer", http.StatusBadRequest)
		return
	}

	err = h.store.GenerateGeneralTicket(count)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate tickets: %v", err), http.StatusInternalServerError)
		return

	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Successfully generated %d general tickets", count),
	})
}

// @Summary Generate a PDF file for general tickets
// @Description Get a PDF file for a single general ticket
// @Tags tickets
// @Security BearerAuth
// @Param id path int true "General ID"
// @Success 200 {file} file "PDF Ticket"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/generate-general/{id} [get]
func (h *Handler) handleGenerateGenerals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid general ID"))
		return
	}

	pdfData, err := h.store.GenerateGeneral(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"ticket.pdf\"")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfData)
}

// @Summary Get general tickets info
// @Description Returns a paginated list of general tickets with their metadata.
// @Tags tickets
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number (default is 1)"
// @Param page_size query int false "Page size (default is 10)"
// @Success 200 {object} types.PaginatedResult[types.GeneralTicket]
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/generals [get]
func (h *Handler) handleGetGeneralsInfo(w http.ResponseWriter, r *http.Request) {
	params := utils.ParsePaginationParams(r)

	paginated, err := h.store.GetGeneralTicketsInfo(params)
	if err != nil {
		log.Printf("❌ Failed to get general tickets: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve general tickets",
		})
		return
	}

	utils.WriteJSON(w, http.StatusOK, paginated)
}

// @Summary Get named tickets info
// @Description Returns a list of named (guest-specific) tickets with QR and PDF data.
// @Tags tickets
// @Security BearerAuth
// @Success 200 {array} types.NamedTicket
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/named [get]
// func (h *Handler) handleGetNamedInfo(w http.ResponseWriter, r *http.Request) {
// 	named, err := h.store.GetNamedTicketsInfo()
// 	if err != nil {
// 		log.Printf("❌ Failed to get named tickets: %v", err)
// 		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{
// 			"error": "Failed to retrieve named tickets",
// 		})
// 		return
// 	}
//
// 	utils.WriteJSON(w, http.StatusOK, named)
// }

// @Summary Get ticket counts
// @Description Returns the total number of named, general, and all tickets.
// @Tags tickets
// @Security BearerAuth
// @Success 200 {object} types.AllTickets
// @Failure 500 {object} types.ErrorResponse
// @Router /tickets/count [get]
func (h *Handler) handleGetTicketsCount(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.GetTicketsCount()
	if err != nil {
		log.Printf("❌ Failed to get ticket counts: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve ticket counts",
		})
		return
	}

	utils.WriteJSON(w, http.StatusOK, counts)
}

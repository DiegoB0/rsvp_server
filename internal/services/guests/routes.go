package guests

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diegob0/rspv_backend/internal/services/auth"
	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type Handler struct {
	store types.GuestStore
}

func NewHandler(store types.GuestStore) *Handler {
	return &Handler{store: store}
}

// Router handler
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Protected routes
	protected := router.PathPrefix("/guests").Subrouter()
	protected.Use(auth.AuthMiddleware)

	// Methods to assing and unassign guests
	protected.HandleFunc("/assign/{guestId}/{tableId}", h.handleAssignGuest).Methods(http.MethodPatch)
	protected.HandleFunc("/unassign/{id}", h.handleUnassignGuest).Methods(http.MethodPatch)

	// Get tickets per guest
	protected.HandleFunc("/tickets/{id}", h.handleGetTicketsPerGuest).Methods(http.MethodGet)

	// Other routes
	protected.HandleFunc("/{id}", h.handleGetGuestByID).Methods(http.MethodGet)
	protected.HandleFunc("/{id}", h.handleDeleteGuest).Methods(http.MethodDelete)
	protected.HandleFunc("/{id}", h.handleUpdateGuest).Methods(http.MethodPatch)
	protected.HandleFunc("", h.handleCreateGuest).Methods(http.MethodPost)
	protected.HandleFunc("", h.handleGetGuests).Methods(http.MethodGet)
}

// @Summary Register a new guest
// @Description Registers a new guset and returns a 201 status on success
// @Tags guests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body types.CreateGuestPayload true "Guest Creation Payload"
// @Success 201 {object} nil
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests [post]
func (h *Handler) handleCreateGuest(w http.ResponseWriter, r *http.Request) {
	// Get JSON paylaod
	var payload types.CreateGuestPayload

	// Show an error if it exists
	if err := utils.ParseJSON(r, &payload); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Validate payload
	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid payload %v", errors))
		return
	}

	// Check if the guest exists
	_, err := h.store.GetGuestByName(payload.FullName)
	if err == nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("guest with name %s already exists", payload.FullName))
		return
	}

	// If not create the guest
	err = h.store.CreateGuest(types.Guest{
		FullName:          payload.FullName,
		Additionals:       *payload.Additionals,
		ConfirmAttendance: *payload.ConfirmAttendance,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, nil)
}

// @Summary Get all guests
// @Description Returns a list of guests
// @Tags guests
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} types.PaginatedResult[types.Guest]/
// @Failure 500 {object} types.ErrorResponse
// @Router /guests [get]
func (h *Handler) handleGetGuests(w http.ResponseWriter, r *http.Request) {
	params := utils.ParsePaginationParams(r)

	guests, err := h.store.GetGuests(params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, guests)
}

// @Summary Get all tickets per guest
// @Description Returns a list of tickets per guest
// @Tags guests
// @Security BearerAuth
// @Produce json
// @Param id path int true "Guest ID"
// @Success 200 {array} types.Guest
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/tickets/{id} [get]
func (h *Handler) handleGetTicketsPerGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guests ID"))
		return
	}

	g, err := h.store.GetTicketsPerGuest(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, g)
}

// @Summary Get guest by ID
// @Description Returns a single guest by their ID
// @Tags guests
// @Security BearerAuth
// @Produce json
// @Param id path int true "Guest ID"
// @Success 200 {object} types.Guest
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/{id} [get]
func (h *Handler) handleGetGuestByID(w http.ResponseWriter, r *http.Request) {
	// Get the params
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guests ID"))
		return
	}

	g, err := h.store.GetGuestByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, g)
}

// @Summary Delete a guest by ID
// @Description Deletes a guest by ID
// @Tags guests
// @Security BearerAuth
// @Param id path int true "Guest ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/{id} [delete]
func (h *Handler) handleDeleteGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guest ID"))
		return
	}

	err = h.store.DeleteGuest(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Summary Update a guest
// @Description Updates guest data by ID (partial update)
// @Tags guests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Guest ID"
// @Param payload body types.UpdateGuestPayload true "Guest fields to update"
// @Success 200 {object} types.Guest
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/{id} [patch]
func (h *Handler) handleUpdateGuest(w http.ResponseWriter, r *http.Request) {
	// Get id
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guest id"))
		return
	}

	// Get the payload
	var payload types.UpdateGuestPayload

	// Validate the payload
	if err := utils.ParseJSON(r, &payload); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Validate payload
	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid payload %v", errors))
		return
	}

	// Get current user from DB if partial update logic is needed
	guest, err := h.store.GetGuestByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Apply updates only if present
	if payload.FullName != nil {
		guest.FullName = *payload.FullName
	}
	if payload.Additionals != nil {
		guest.Additionals = *payload.Additionals
	}
	if payload.ConfirmAttendance != nil {
		guest.ConfirmAttendance = *payload.ConfirmAttendance
	}

	if err := h.store.UpdateGuest(guest); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, guest)
}

// @Summary Assign a guest to a table
// @Description Updates guest data by ID (partial update)
// @Tags guests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param guestId path int true "Guest ID"
// @Param tableId path int true "Table ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/assign/{guestId}/{tableId} [patch]
func (h *Handler) handleAssignGuest(w http.ResponseWriter, r *http.Request) {
	// Get the guestId
	vars := mux.Vars(r)
	guestIdStr := vars["guestId"]
	guestId, err := strconv.Atoi(guestIdStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guest id"))
		return
	}

	tableIdStr := vars["tableId"]
	tableId, err := strconv.Atoi(tableIdStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid table id"))
		return
	}

	if err := h.store.AssignGuest(guestId, tableId); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

// @Summary Unassign a guest to a table
// @Description Updates guest data by ID (partial update)
// @Tags guests
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Guest ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /guests/unassign/{id} [patch]
func (h *Handler) handleUnassignGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid guest id"))
		return
	}

	if err := h.store.UnassignGuest(id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

package tables

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
	store types.TableStore
}

func NewHandler(store types.TableStore) *Handler {
	return &Handler{store: store}
}

// Router handler
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Protected routes
	protected := router.PathPrefix("/tables").Subrouter()
	protected.Use(auth.AuthMiddleware)

	// Methods with join tables
	protected.HandleFunc("/guests", h.handleGetTablesAndGuests).Methods(http.MethodGet)
	protected.HandleFunc("/guests/{id}", h.handleGetTableAndGuestsByID).Methods(http.MethodGet)

	// Other routes
	protected.HandleFunc("", h.handleCreateTable).Methods(http.MethodPost)
	protected.HandleFunc("", h.handleGetTables).Methods(http.MethodGet)
	protected.HandleFunc("/{id}", h.handleGetTableByID).Methods(http.MethodGet)
	protected.HandleFunc("/{id}", h.handleDeleteTable).Methods(http.MethodDelete)
	protected.HandleFunc("/{id}", h.handleUpateTable).Methods(http.MethodPatch)
}

// @Summary Register a new table
// @Description Registers a new table and returns a 201 status on success
// @Tags mesas
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body types.CreateTablePayload true "Registration Payload"
// @Success 201 {object} nil
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tables [post]
func (h *Handler) handleCreateTable(w http.ResponseWriter, r *http.Request) {
	// Get JSON paylaod
	var payload types.CreateTablePayload

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

	// Check if the table exists
	_, err := h.store.GetTableByName(payload.Name)
	if err == nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("table with name %s already exists", payload.Name))
		return
	}

	// If not create the table
	err = h.store.CreateTable(types.Table{
		Name:     payload.Name,
		Capacity: payload.Capacity,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, nil)
}

// @Summary Get all tables
// @Description Returns a paginated list of tables
// @Tags mesas
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param search query string false "Search term to filter tables by name"
// @Success 200 {object} types.PaginatedResult[types.Table]
// @Failure 500 {object} types.ErrorResponse
// @Router /tables [get]
func (h *Handler) handleGetTables(w http.ResponseWriter, r *http.Request) {
	params := utils.ParsePaginationParams(r)

	paginated, err := h.store.GetTables(params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, paginated)
}

// @Summary Get tables by ID
// @Description Returns a single table by their ID
// @Tags mesas
// @Security BearerAuth
// @Produce json
// @Param id path int true "Table ID"
// @Success 200 {object} types.Table
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tables/{id} [get]
func (h *Handler) handleGetTableByID(w http.ResponseWriter, r *http.Request) {
	// Get the params
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid user ID"))

		return
	}

	t, err := h.store.GetTableByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, t)
}

// @Summary Delete a table by ID
// @Description Deletes a table by ID
// @Tags mesas
// @Security BearerAuth
// @Param id path int true "Table ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tables/{id} [delete]
func (h *Handler) handleDeleteTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid table ID"))

		return
	}

	err = h.store.DeleteTable(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Summary Update a table
// @Description Updates table data by ID (partial update)
// @Tags mesas
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Table ID"
// @Param payload body types.UpdateTablePayload true "Table fields to update"
// @Success 200 {object} types.Table
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tables/{id} [patch]
func (h *Handler) handleUpateTable(w http.ResponseWriter, r *http.Request) {
	// Get id
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid table id"))
		return
	}

	// Get the payload
	var payload types.UpdateTablePayload

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
	table, err := h.store.GetTableByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Apply updates only if present
	if payload.Name != nil {
		table.Name = *payload.Name
	}
	if payload.Capacity != nil {
		table.Capacity = *payload.Capacity
	}

	if err := h.store.UpdateTable(table); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, table)
}

// @Summary Get tables and guests related (paginated)
// @Description Returns a paginated list of tables with guests and generals
// @Tags mesas
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param search query string false "Search term to filter tables by name"
// @Success 200 {object} types.PaginatedResult[types.TableAndGuests]
// @Failure 500 {object} types.ErrorResponse
// @Router /tables/guests [get]
func (h *Handler) handleGetTablesAndGuests(w http.ResponseWriter, r *http.Request) {
	params := utils.ParsePaginationParams(r)

	result, err := h.store.GetTablesWithGuests(params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

// @Summary Get table with guests by ID
// @Description Returns a single table with guests by their ID
// @Tags mesas
// @Security BearerAuth
// @Produce json
// @Param id path int true "Table ID"
// @Success 200 {object} types.TableAndGuests
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /tables/guests/{id} [get]
func (h *Handler) handleGetTableAndGuestsByID(w http.ResponseWriter, r *http.Request) {
	// Get the params
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid table ID"))
		return
	}

	t, err := h.store.GetTableWithGuestsByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, t)
}

package generals

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
	store types.GeneralStore
}

func NewHandler(store types.GeneralStore) *Handler {
	return &Handler{store: store}
}

// Router handler
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Protected routes
	protected := router.PathPrefix("/generals").Subrouter()
	protected.Use(auth.AuthMiddleware)

	// Methods to assing and unassign guests
	protected.HandleFunc("/assign/{generalId}/{tableId}", h.handleAssignGuest).Methods(http.MethodPatch)
	protected.HandleFunc("/unassign/{id}", h.handleUnassignGuest).Methods(http.MethodPatch)

	// Other routes
	protected.HandleFunc("/{id}", h.handleDeleteGuest).Methods(http.MethodDelete)
}

// @Summary Assign a general to a table
// @Description Assign a general to a table
// @Tags generals
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param generalId path int true "General ID"
// @Param tableId path int true "Table ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /generals/assign/{generalId}/{tableId} [patch]
func (h *Handler) handleAssignGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	generalIdStr := vars["generalId"]
	generalId, err := strconv.Atoi(generalIdStr)
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

	if err := h.store.AssignGeneral(generalId, tableId); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

// @Summary Unassign a general to a table
// @Description Unassign general ticket from a table
// @Tags generals
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "General ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /generals/unassign/{id} [patch]
func (h *Handler) handleUnassignGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid general id"))
		return
	}

	if err := h.store.UnassignGeneral(id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

// @Summary Delete a general by ID
// @Description Deletes a general ticket by ID
// @Tags generals
// @Security BearerAuth
// @Param id path int true "General ID"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /generals/{id} [delete]
func (h *Handler) handleDeleteGuest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid general ID"))
		return
	}

	err = h.store.DeleteGeneral(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

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
	protected.HandleFunc("/assign/{generalId}/{tableId}", h.handleAssignGeneral).Methods(http.MethodPatch)
	protected.HandleFunc("/unassign/{id}", h.handleUnassignGeneral).Methods(http.MethodPatch)

	// Other routes
	protected.HandleFunc("", h.handleDeleteLastGenerals).Methods(http.MethodDelete)
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
func (h *Handler) handleAssignGeneral(w http.ResponseWriter, r *http.Request) {
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
func (h *Handler) handleUnassignGeneral(w http.ResponseWriter, r *http.Request) {
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

// @Summary Delete last N generals
// @Description Deletes the last N general tickets in queue order (only unassigned allowed)
// @Tags generals
// @Security BearerAuth
// @Param count query int false "Number of generals to delete (default is 1)"
// @Success 204 "No content"
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /generals [delete]
func (h *Handler) handleDeleteLastGenerals(w http.ResponseWriter, r *http.Request) {
	// Parse optional ?count query param
	countStr := r.URL.Query().Get("count")
	count := 1 // default

	if countStr != "" {

		parsed, err := strconv.Atoi(countStr)
		if err != nil || parsed <= 0 {
			utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid count"))

			return
		}
		count = parsed
	}

	// Call store method

	err := h.store.DeleteLastGenerals(count)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err) // use 400 for known issues like assigned generals
		return

	}

	utils.WriteJSON(w, http.StatusNoContent, nil)
}

package user

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/diegob0/rspv_backend/internal/config"
	"github.com/diegob0/rspv_backend/internal/services/auth"
	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type Handler struct {
	store types.UserStore
}

func NewHandler(store types.UserStore) *Handler {
	return &Handler{store: store}
}

// Router handler
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/login", h.handleLogin).Methods(http.MethodPost)
	// router.HandleFunc("/register", h.handleRegister).Methods(http.MethodPost)

	// Protected routes
	protected := router.PathPrefix("/users").Subrouter()
	protected.Use(auth.AuthMiddleware)

	protected.HandleFunc("", h.handleGetUsers).Methods(http.MethodGet)
	protected.HandleFunc("/me", h.handleGetUserByEmail).Methods(http.MethodGet)
	protected.HandleFunc("/{id}", h.handleGetUserByID).Methods(http.MethodGet)
	protected.HandleFunc("", h.handleRegister).Methods(http.MethodPost)
	protected.HandleFunc("/{id}", h.handleDeleteUsers).Methods(http.MethodDelete)
	protected.HandleFunc("/{id}", h.handleUpdateUsers).Methods(http.MethodPatch)
}

// @Summary Login
// @Description Authenticates a user and returns a JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body types.LoginUserPayload true "Login Payload"
// @Success 200 {object} types.LoginSuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /login [post]
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Get JSON payload
	var payload types.LoginUserPayload

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

	// Find the user by email
	u, err := h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid credentials"))
		return
	}

	// Compare passwords
	if !auth.ComparePasswords(u.Password, []byte(payload.Password)) {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid credentials"))
		return
	}

	// Generate the JWT
	secret := []byte(config.Envs.JWTSecret)
	token, err := auth.CreateJWT(secret, u.ID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

// @Summary Register a new user
// @Description Registers a new user and returns a 201 status on success
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body types.RegisterUserPayload true "Registration Payload"
// @Success 201 {object} nil
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users [post]
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Get JSON paylaod
	var payload types.RegisterUserPayload

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

	// Check if the user exists
	_, err := h.store.GetUserByEmail(payload.Email)
	if err == nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("user with email %s already exists", payload.Email))
		return
	}

	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// If not create the user
	err = h.store.CreateUser(types.User{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		Password:  hashedPassword,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, nil)
}

// @Summary Get all users
// @Description Returns a list of users
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {array} types.User
// @Failure 500 {object} types.ErrorResponse
// @Router /users [get]
func (h *Handler) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	u, err := h.store.GetUsers()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, u)
}

// @Summary Get user by ID
// @Description Returns a single user by their ID
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} types.User
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users/{id} [get]
func (h *Handler) handleGetUserByID(w http.ResponseWriter, r *http.Request) {
	// Get the params
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Parse the id
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid user ID"))

		return
	}

	u, err := h.store.GetUserByID(id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, u)
}

// @Summary Get user by email
// @Description Returns a user by email (requires JSON body with email)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body types.GetUserByEmailPayload true "Email Payload"
// @Success 200 {object} types.User
// @Failure 400 {object} types.ErrorResponse
// @Router /users/me [get]
func (h *Handler) handleGetUserByEmail(w http.ResponseWriter, r *http.Request) {
	// Get JSON payload
	var payload types.GetUserByEmailPayload

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

	// Find the user by email
	u, err := h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("no user found"))
		return
	}

	utils.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) handleDeleteUsers(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) handleUpdateUsers(w http.ResponseWriter, r *http.Request) {}

package api

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/diegob0/rspv_backend/docs"
	"github.com/diegob0/rspv_backend/internal/services/user"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type APIServer struct {
	addr string
	db   *sql.DB
}

func NewAPIServer(addr string, db *sql.DB) *APIServer {
	return &APIServer{
		addr: addr,
		db:   db,
	}
}

func (s *APIServer) Run() error {
	router := mux.NewRouter()

	// Routes for swagger
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Main API routes
	subrouter := router.PathPrefix("/api/v1").Subrouter()

	// Register each service
	userStore := user.NewStore(s.db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(subrouter)

	log.Println("Listening on port", s.addr)

	return http.ListenAndServe(s.addr, router)
}

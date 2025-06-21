package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/diegob0/rspv_backend/docs"
	"github.com/diegob0/rspv_backend/internal/services/guests"
	"github.com/diegob0/rspv_backend/internal/services/tables"
	"github.com/diegob0/rspv_backend/internal/services/tickets"
	"github.com/diegob0/rspv_backend/internal/services/user"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

	// Users routes
	userStore := user.NewStore(s.db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(subrouter)

	// Tables routes
	tableStore := tables.NewStore(s.db)
	tableHandler := tables.NewHandler(tableStore)
	tableHandler.RegisterRoutes(subrouter)

	// Guests routes
	guestStore := guests.NewStore(s.db)
	guestHandler := guests.NewHandler(guestStore)
	guestHandler.RegisterRoutes(subrouter)

	ticketStore := tickets.NewStore(s.db)
	ticketHandler := tickets.NewHandler(ticketStore)
	ticketHandler.RegisterRoutes(subrouter)

	log.Println("Listening on port", s.addr)

	// Cors config for dev and prod
	prod := strings.ToLower(os.Getenv("PROD")) == "true"

	var allowedOrigins []string
	if prod {
		allowedOrigins = []string{
			"https://vaneycarlos.com",
			"https://www.vaneycarlos.com",
		}
	} else {
		allowedOrigins = []string{
			"*",
		}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	handler := c.Handler(router)

	return http.ListenAndServe(s.addr, handler)
}

// @title RSVP API
// @version 1.0
// @description API for RSVP Backend
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Provide your JWT token like: Bearer <token>

// ------- Order Tags --------

// @tag.name auth
// @tag.description Authentication operations

// @tag.name users
// @tag.description User management

// @tag.name mesas
// @tag.description Table management

// @tag.name guests
// @tag.description Guest management

// @tag.name generals
// @tag.description Generals management

// @tag.name tickets
// @tag.description Tickets management
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/diegob0/rspv_backend/cmd/api"
	"github.com/diegob0/rspv_backend/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Hello World")
	database, err := db.ConnectToDB()
	if err != nil {
		log.Fatal("Failed to connect to DB", err)
	}

	log.Println("✅ Connected to the database successfully")

	server := api.NewAPIServer(":"+port, database)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}

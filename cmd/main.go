package main

import (
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

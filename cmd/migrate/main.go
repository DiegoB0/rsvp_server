package main

import (
	"log"
	"os"

	"github.com/diegob0/rspv_backend/internal/db"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go [up|down]")
	}

	action := os.Args[1]

	dbConn, err := db.ConnectToDB()
	if err != nil {
		log.Fatal("❌ Failed to connect to DB:", err)
	}

	driver, err := postgres.WithInstance(dbConn, &postgres.Config{})
	if err != nil {
		log.Fatal("❌ Could not create migration driver:", err)
	}

	m, err := migrate.NewWithDatabaseInstance(

		"file://cmd/migrate/migrations",

		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("❌ Failed to create migrate instance:", err)
	}

	switch action {

	case "up":
		log.Println("🔼 Applying migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal("❌ Migration up error:", err)
		}
		log.Println("✅ Migrations applied")

	case "down":

		log.Println("⏬ Rolling back last migration...")
		if err := m.Steps(-1); err != nil {
			log.Fatal("❌ Migration down error:", err)
		}
		log.Println("✅ Rolled back successfully")

	default:
		log.Fatalf("Unknown action: %s. Use 'up' or 'down'.", action)
	}
}

package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diegob0/rspv_backend/internal/config"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func ConnectToDB() (*sql.DB, error) {
	_ = godotenv.Load()

	host := config.Envs.DBHost
	port := config.Envs.DBPort
	user := config.Envs.DBUser
	password := config.Envs.DBPassword
	dbname := config.Envs.DBName

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return nil, fmt.Errorf("one or more DB env vars are missing")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening DB: %w", err)
	}

	// Set max open connections
	db.SetMaxOpenConns(50)

	// Max idle connections
	db.SetMaxIdleConns(25)

	// Recycle connections every 5 min
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to DB: %w", err)
	}

	return db, nil
}

package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diegob0/rspv_backend/internal/db"
	"github.com/diegob0/rspv_backend/internal/services/jobs"
	"github.com/diegob0/rspv_backend/internal/services/jobs/queue"
	"github.com/diegob0/rspv_backend/internal/services/tickets"
)

func main() {
	log.Println("🚀 Starting worker...")

	// Connect to DB using your existing logic
	database, err := db.ConnectToDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}
	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			log.Printf("⚠️ Failed to close DB: %v", err)
		}
	}(database)

	store := tickets.NewStore(database)

	// Graceful shutdown

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("🛑 Shutdown signal received")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("👋 Worker shutting down")
			return
		default:
		}

		// Block until job is available
		payload, err := queue.DequeueJob(ctx, queue.QrJobQueue)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("👋 Context canceled, exiting...")
				return
			}
			log.Printf("⚠️ Error dequeuing job: %v", err)
			continue
		}

		if payload == "" {
			time.Sleep(1 * time.Second)
			continue
		}

		var job queue.QrUploadJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("⚠️ Failed to unmarshal job payload: %v", err)
			continue
		}

		log.Printf("🔧 Processing job for ticket ID: %d", job.TicketID)

		qrBytes := make([][]byte, 0, len(job.QrCodes))
		for _, qrStr := range job.QrCodes {
			data, err := base64.StdEncoding.DecodeString(qrStr)
			if err != nil {
				log.Printf("❌ Failed to decode QR code string: %v", err)
				continue

			}
			qrBytes = append(qrBytes, data)
		}

		if err := jobs.UploadQrCodes(job.TicketID, qrBytes, store); err != nil {
			log.Printf("❌ Failed to upload QR codes for ticket %d: %v", job.TicketID, err)
		} else {
			log.Printf("✅ Job for ticket %d processed successfully", job.TicketID)
		}
	}
}

package main

import (
	"context"
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

	database, err := db.ConnectToDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}
	defer database.Close()

	store := tickets.NewStore(database)

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("🛑 Shutdown signal received")

		cancel()
	}()

	jobQueues := []string{queue.QrJobQueue, queue.PdfJobQueue}

	for {
		select {

		case <-ctx.Done():
			log.Println("👋 Worker shutting down")
			return
		default:

		}

		for _, queueName := range jobQueues {

			payload, err := queue.DequeueJob(ctx, queueName)
			if err != nil {
				if ctx.Err() != nil {
					log.Println("👋 Context canceled, exiting...")

					return
				}

				log.Printf("⚠️ Error dequeuing job from %s: %v", queueName, err)

				continue

			}

			if payload == "" {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			switch queueName {

			case queue.QrJobQueue:

				var job queue.QrUploadJob
				if err := json.Unmarshal([]byte(payload), &job); err != nil {

					log.Printf("⚠️ Failed to unmarshal QR job: %v", err)
					continue
				}

				log.Printf("🔧 Processing QR job for ticket ID: %d", job.TicketID)

				qrBytes := make([][]byte, 0, len(job.QrCodes))
				for _, qrStr := range job.QrCodes {
					data, err := base64.StdEncoding.DecodeString(qrStr)
					if err != nil {
						log.Printf("❌ Failed to decode QR string: %v", err)
						continue
					}

					qrBytes = append(qrBytes, data)
				}

				if err := jobs.UploadQrCodes(job.TicketID, qrBytes, store); err != nil {
					log.Printf("❌ Failed to upload QR codes: %v", err)
				} else {
					log.Printf("✅ QR job for ticket %d processed", job.TicketID)
				}

			case queue.PdfJobQueue:
				var job queue.PdfUploadJob
				if err := json.Unmarshal([]byte(payload), &job); err != nil {
					log.Printf("⚠️ Failed to unmarshal PDF job: %v", err)
					continue
				}

				log.Printf("🧾 Processing PDF job for ticket ID: %d", job.TicketID)

				pdfBytes, err := base64.StdEncoding.DecodeString(job.PDFBase64)
				if err != nil {
					log.Printf("❌ Failed to decode PDF base64: %v", err)
					continue
				}

				if err := jobs.UploadPDF(job.TicketID, pdfBytes, store); err != nil {
					log.Printf("❌ Failed to upload PDF: %v", err)
				} else {
					log.Printf("✅ PDF job for ticket %d processed", job.TicketID)
				}
			}
		}
	}
}

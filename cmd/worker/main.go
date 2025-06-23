package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/diegob0/rspv_backend/internal/db"
	"github.com/diegob0/rspv_backend/internal/services/jobs"
	"github.com/diegob0/rspv_backend/internal/services/jobs/queue"
	"github.com/diegob0/rspv_backend/internal/services/tickets"
)

func main() {
	log.Println("Starting worker...")

	database, err := db.ConnectToDB()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()

	store := tickets.NewStore(database)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
	}()

	jobQueues := []string{queue.QrJobQueue, queue.PdfJobQueue, queue.EmailJobQueue, queue.FullUploadQueue}

	var wg sync.WaitGroup
	for _, queueName := range jobQueues {
		queueName := queueName
		wg.Add(1)

		go startWorker(ctx, &wg, queueName, store)
	}

	wg.Wait()
	log.Println("All workers exited. Goodbye!")
}

func startWorker(ctx context.Context, wg *sync.WaitGroup, queueName string, store *tickets.Store) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s worker shutting down", queueName)
			return
		default:
		}

		payload, err := queue.DequeueJob(ctx, queueName)
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Context canceled for %s, exiting...", queueName)
				return
			}
			log.Printf("Error dequeuing job from %s: %v", queueName, err)
			continue
		}

		if payload == "" {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		processJob(ctx, queueName, payload, store)
	}
}

func processJob(ctx context.Context, queueName, payload string, store *tickets.Store) {
	switch queueName {

	case queue.QrJobQueue:
		var job queue.QrUploadJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("⚠️ Failed to unmarshal QR job: %v", err)
			return
		}
		log.Printf("Processing QR job for ticket ID: %d", job.TicketID)

		qrBytes := make([][]byte, 0, len(job.QrCodes))
		for _, qrStr := range job.QrCodes {
			data, err := base64.StdEncoding.DecodeString(qrStr)
			if err != nil {
				log.Printf("Failed to decode QR string: %v", err)
				continue
			}
			qrBytes = append(qrBytes, data)
		}

		retry(ctx, int64(job.TicketID), func() error {
			return jobs.UploadQrCodes(job.TicketID, qrBytes, job.TicketType, store)
		}, "QR")

		// PDF queue
	case queue.PdfJobQueue:

		var job queue.PdfUploadJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("Failed to unmarshal PDF job: %v", err)
			return
		}
		log.Printf("Processing PDF job for ticket ID: %d", job.TicketID)

		pdfBytes, err := base64.StdEncoding.DecodeString(job.PDFBase64)
		if err != nil {
			log.Printf("Failed to decode PDF base64: %v", err)
			return

		}

		retry(ctx, int64(job.TicketID), func() error {
			return jobs.UploadPDF(job.TicketID, pdfBytes, job.TicketType, store)
		}, "PDF")

		// EMAIL queue
	case queue.EmailJobQueue:
		var job queue.EmailSendJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("Failed to unmarshal Email job: %v", err)
			return
		}
		log.Printf("Processing Email job for ticket ID: %d", job.GuestID)

		pdfBytes, err := downloadPDF(job.PDFURL)
		if err != nil {
			log.Printf("Failed to download pdf from %s: %v", job.PDFURL, err)
		}

		retry(ctx, int64(job.GuestID), func() error {
			return jobs.SendTicketEmailWithPdf(job.GuestID, job.Recipient, pdfBytes, store)
		}, "Email")

	case queue.FullUploadQueue:
		var job queue.FullUploadJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			log.Printf("Failed to unmarshal FullUpload job: %v", err)
			return
		}
		log.Printf("Processing FullUpload job for ticket ID: %d", job.TicketID)

		qrBytes := make([][]byte, 0, len(job.QrCodes))
		for _, qrStr := range job.QrCodes {
			data, err := base64.StdEncoding.DecodeString(qrStr)
			if err != nil {
				log.Printf("Failed to decode QR string: %v", err)
				continue
			}
			qrBytes = append(qrBytes, data)
		}

		pdfBytes, err := base64.StdEncoding.DecodeString(job.PDFBase64)
		if err != nil {
			log.Printf("Failed to decode PDF base64: %v", err)
			return
		}

		retry(ctx, int64(job.TicketID), func() error {
			if err := jobs.UploadQrCodes(job.TicketID, qrBytes, job.TicketType, store); err != nil {
				return err
			}

			return jobs.UploadPDF(job.TicketID, pdfBytes, job.TicketType, store)
		}, "FullUpload")

	}
}

func retry(ctx context.Context, ticketID int64, task func() error, label string) {
	const maxRetries = 5
	delay := 500 * time.Millisecond

	for i := 1; i <= maxRetries; i++ {
		err := task()
		if err == nil {
			log.Printf("%s job for ticket %d processed on attempt %d", label, ticketID, i)
			return
		}

		log.Printf("Retry %d/%d for %s job (ticket ID: %d): %v", i, maxRetries, label, ticketID, err)

		select {
		case <-ctx.Done():
			log.Printf("%s job retry canceled for ticket %d", label, ticketID)
			return
		case <-time.After(delay):
		}

		delay *= 2
	}

	log.Printf("%s job for ticket %d failed after %d retries", label, ticketID, maxRetries)
}

// Helper functions
func downloadPDF(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

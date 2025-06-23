package jobs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/diegob0/rspv_backend/internal/services/aws"
	"github.com/diegob0/rspv_backend/internal/services/tickets"
)

func SendTicketEmailWithPdf(guestID int, recipientEmail string, pdfFile []byte, store *tickets.Store) error {
	ctx := context.Background()

	subject := os.Getenv("EMAIL_SUBJECT")
	if subject == "" {
		subject = "Entrada para la boda"
	}

	bodyText := os.Getenv("EMAIL_BODY_TEXT")
	if bodyText == "" {
		bodyText = "Muchas gracias por confirmar tu asistencia!"
	}

	emailer, err := aws.NewSESEmailer()
	if err != nil {
		return fmt.Errorf("failed to init SES emailer: %w", err)
	}

	err = emailer.SendEmailWithAttachment(ctx, recipientEmail, subject, bodyText, pdfFile, fmt.Sprintf("ticket-%d.pdf", guestID))
	if err != nil {
		return fmt.Errorf("failed to send SES email: %w", err)
	}

	log.Printf("Sent email with PDF attachment to %s for ticket %d", recipientEmail, guestID)
	return nil
}

func UploadQrCodes(ticketID int, qrCodes [][]byte, ticketType string, store *tickets.Store) error {
	ctx := context.Background()

	uploader, err := aws.NewS3Uploader()
	if err != nil {
		return fmt.Errorf("failed to init uploader: %w", err)
	}

	var urls []string
	for i, qr := range qrCodes {
		key := fmt.Sprintf("qr-codes/ticket-%d-%d.png", ticketID, i)
		url, err := uploader.UploadBytes(ctx, key, qr, "image/png")
		if err != nil {
			log.Printf("failed to upload qr code %d: %v", i, err)
			continue
		}
		urls = append(urls, url)
	}

	// Check what's inside the urls
	log.Printf("urls slice before saving: %#v", urls)
	time.Sleep(1 * time.Second)

	if len(urls) > 0 {
		updateFn := func() error {
			if ticketType == "general" {
				return store.UpdateGeneralQrCodeUrls(ticketID, urls)
			} else {
				return store.UpdateQrCodeUrls(ticketID, urls)
			}
		}

		if err := retry(updateFn, 3, 2*time.Second); err != nil {
			return fmt.Errorf("failed to save QR code URLs after retries: %w", err)
		}
	}
	return nil
}

func UploadPDF(ticketID int, pdfFile []byte, ticketType string, store *tickets.Store) error {
	ctx := context.Background()

	uploader, err := aws.NewS3Uploader()
	if err != nil {
		return fmt.Errorf("failed to init uploader: %w", err)
	}

	key := fmt.Sprintf("pdf-files/ticket-%d.pdf", ticketID)
	url, err := uploader.UploadBytes(ctx, key, pdfFile, "application/pdf")
	if err != nil {
		return fmt.Errorf("failed to upload PDF file: %w", err)
	}

	log.Printf("Uploaded PDF URL: %s", url)

	updateFn := func() error {
		if ticketType == "general" {
			return store.UpdateGeneralPDFfileUrls(ticketID, url)
		} else {
			return store.UpdatePDFfileUrls(ticketID, url)
		}
	}

	if err := retry(updateFn, 3, 2*time.Second); err != nil {
		return fmt.Errorf("failed to save PDF URL after retries: %w", err)
	}

	return nil
}

func retry(fn func() error, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		log.Printf("Retry %d/%d failed: %v", i+1, attempts, err)
		time.Sleep(delay)
	}
	return fmt.Errorf("all retries failed")
}

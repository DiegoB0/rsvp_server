package jobs

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/diegob0/rspv_backend/internal/services/aws"
	"github.com/diegob0/rspv_backend/internal/services/tickets"
)

func SendTicketEmailWithPdf(ticketID int, recipientEmail string, pdfFile []byte, store *tickets.Store) error {
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

	err = emailer.SendEmailWithAttachment(ctx, recipientEmail, subject, bodyText, pdfFile, fmt.Sprintf("ticket-%d.pdf", ticketID))
	if err != nil {
		return fmt.Errorf("failed to send SES email: %w", err)
	}

	log.Printf("Sent email with PDF attachment to %s for ticket %d", recipientEmail, ticketID)
	return nil
}

func UploadQrCodes(ticketID int, qrCodes [][]byte, store *tickets.Store) error {
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

	if len(urls) > 0 {
		if err := store.UpdateQrCodeUrls(ticketID, urls); err != nil {
			return fmt.Errorf("failed to save QR code URLs: %w", err)
		}
	}

	return nil
}

func UploadPDF(ticketID int, pdfFile []byte, store *tickets.Store) error {
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

	if err := store.UpdatePDFfileUrls(ticketID, []string{url}); err != nil {
		return fmt.Errorf("failed to save PDF file URLs: %w", err)
	}

	return nil
}

package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	sesv2 "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type SESEmailer struct {
	client *sesv2.Client
	sender string
}

func NewSESEmailer() (*SESEmailer, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := sesv2.NewFromConfig(cfg)
	sender := os.Getenv("AWS_SES_SENDER")
	if sender == "" {
		return nil, fmt.Errorf("AWS_SES_SENDER is not set")
	}

	return &SESEmailer{
		client: client,
		sender: sender,
	}, nil
}

func (s *SESEmailer) SendEmailWithAttachment(ctx context.Context, recipient, subject, bodyText string, pdfData []byte, filename string) error {
	var emailRaw bytes.Buffer
	writer := multipart.NewWriter(&emailRaw)

	boundary := writer.Boundary()

	// Build email headers manually
	fmt.Fprintf(&emailRaw, "From: %s\r\n", s.sender)
	fmt.Fprintf(&emailRaw, "To: %s\r\n", recipient)
	fmt.Fprintf(&emailRaw, "Subject: %s\r\n", subject)
	fmt.Fprintf(&emailRaw, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&emailRaw, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary)
	fmt.Fprintf(&emailRaw, "\r\n--%s\r\n", boundary)

	// Plain text part
	fmt.Fprintf(&emailRaw, "Content-Type: text/plain; charset=utf-8\r\n")
	fmt.Fprintf(&emailRaw, "Content-Transfer-Encoding: 7bit\r\n\r\n")
	fmt.Fprintf(&emailRaw, "%s\r\n", bodyText)
	fmt.Fprintf(&emailRaw, "\r\n--%s\r\n", boundary)

	// Attachment part
	fmt.Fprintf(&emailRaw, "Content-Type: application/pdf\r\n")
	fmt.Fprintf(&emailRaw, "Content-Disposition: attachment; filename=\"%s\"\r\n", filename)
	fmt.Fprintf(&emailRaw, "Content-Transfer-Encoding: base64\r\n\r\n")

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(pdfData)))
	base64.StdEncoding.Encode(encoded, pdfData)
	emailRaw.Write(encoded)

	fmt.Fprintf(&emailRaw, "\r\n--%s--\r\n", boundary)

	// Send with SES
	input := &sesv2.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: emailRaw.Bytes(),
		},
	}

	_, err := s.client.SendRawEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send SES email: %w", err)
	}

	return nil
}

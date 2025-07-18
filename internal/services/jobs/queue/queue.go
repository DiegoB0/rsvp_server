package queue

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Type of queues
const QrJobQueue = "qr_upload_jobs"

const SingleQrJobQueue = "single_qr_upload_job"

const PdfJobQueue = "pdf_upload_jobs"

const EmailJobQueue = "email_job_queue"

const FullUploadQueue = "full_upload_jobs"

// Structs for each queue
type QrUploadJob struct {
	TicketID   int      `json:"ticketID"`
	QrCodes    []string `json:"qrCodes"`
	TicketType string   `json:"ticketType"`
}

type PdfUploadJob struct {
	TicketID   int    `json:"ticketID"`
	PDFBase64  string `json:"pdfBase64"`
	TicketType string `json:"ticketType"`
}

type EmailSendJob struct {
	GuestID   int    `json:"guest_id"`
	Recipient string `json:"recipient"`
	PDFURL    string `json:"pdf_url"`
}

type FullUploadJob struct {
	TicketID   int      `json:"ticketID"`
	TicketType string   `json:"ticketType"`
	QrCodes    []string `json:"qrCodes"`
	PDFBase64  string   `json:"pdfBase64"`
}

var (
	redisClient *redis.Client
	once        sync.Once
)

func getRedisClient() *redis.Client {
	once.Do(func() {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})
	})
	return redisClient
}

func EnqueueJob(ctx context.Context, queueName string, jobPayload string) error {
	client := getRedisClient()
	return client.LPush(ctx, queueName, jobPayload).Err()
}

func DequeueJob(ctx context.Context, queueName string) (string, error) {
	client := getRedisClient()

	result, err := client.BRPop(ctx, time.Second, queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

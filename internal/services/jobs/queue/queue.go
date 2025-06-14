package queue

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const QrJobQueue = "qr_upload_jobs"

type QrUploadJob struct {
	TicketID int      `json:"ticketID"`
	QrCodes  []string `json:"qrCodes"`
}

// PDF files
const PdfJobQueue = "pdf_upload_jobs"

type PdfUploadJob struct {
	TicketID  int    `json:"ticketID"`
	PDFBase64 string `json:"pdfFiles"`
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

package queue

import (
	"context"
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Types for the job queue
const QrJobQueue = "qr_upload_jobs"

type QrUploadJob struct {
	TicketID int      `json:"ticketID"`
	QrCodes  []string `json:"qrCodes"`
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
	result, err := client.BRPop(ctx, 0, queueName).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}

	return result[1], nil
}

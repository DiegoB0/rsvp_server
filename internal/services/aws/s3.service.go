package aws

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Uploader struct {
	client *s3.Client
	bucket string
}

func NewS3Uploader() (*S3Uploader, error) {
	// Load AWS config from environment variables or shared config files
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	bucket := os.Getenv("AWS_S3_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET is not set")
	}

	return &S3Uploader{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *S3Uploader) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to s3: %w", err)
	}

	// Return the full s3 URL (you can customize for your bucket and region)
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, os.Getenv("AWS_REGION"), key)
	return url, nil
}

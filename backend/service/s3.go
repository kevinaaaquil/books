package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Service struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3Service(ctx context.Context, bucket, region, accessKeyID, secretAccessKey string) (*S3Service, error) {
	if bucket == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET is required")
	}
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if accessKeyID != "" && secretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &S3Service{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		region: region,
	}, nil
}

// Upload stores the file in S3 under prefix (e.g. "user-id/"). Returns the object key.
func (s *S3Service) Upload(ctx context.Context, prefix, originalFilename string, body io.Reader, contentType string) (string, error) {
	ext := filepath.Ext(originalFilename)
	key := prefix + uuid.New().String() + ext
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

// Delete removes the object from S3.
func (s *S3Service) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetObject downloads the object from S3 and returns its body and content type. Caller must close the returned reader.
func (s *S3Service) GetObject(ctx context.Context, key string) (body io.ReadCloser, contentType string, err error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return out.Body, ct, nil
}

// PresignedGetURL returns a temporary URL to download the object (e.g. for reading the book).
// If responseFilename is non-empty, the presigned URL will set ResponseContentDisposition
// so the browser uses that name instead of the S3 key when saving the file.
func (s *S3Service) PresignedGetURL(ctx context.Context, key string, expiry time.Duration, responseFilename string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if responseFilename != "" {
		// Sanitize for Content-Disposition: escape \ and ", then quote
		safe := responseFilename
		safe = strings.ReplaceAll(safe, "\\", "\\\\")
		safe = strings.ReplaceAll(safe, "\"", "\\\"")
		input.ResponseContentDisposition = aws.String(`attachment; filename="` + safe + `"`)
	}
	presigner := s3.NewPresignClient(s.client)
	req, err := presigner.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

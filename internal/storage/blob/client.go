package blob

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	bucket        string
	client        *s3.Client
	presignClient *s3.PresignClient
	uploadTTL     time.Duration
	downloadTTL   time.Duration
}

type PresignedRequest struct {
	URL     string
	Method  string
	Headers map[string][]string
	Expires time.Time
}

type Config struct {
	Endpoint           string
	Region             string
	Bucket             string
	AccessKeyID        string
	SecretAccessKey    string
	ForcePathStyle     bool
	PresignUploadTTL   time.Duration
	PresignDownloadTTL time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("storage bucket is required")
	}
	if cfg.Region == "" {
		return nil, errors.New("storage region is required")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("storage credentials are required")
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &Client{
		bucket:        cfg.Bucket,
		client:        s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		uploadTTL:     cfg.PresignUploadTTL,
		downloadTTL:   cfg.PresignDownloadTTL,
	}, nil
}

func (c *Client) PresignPutObject(ctx context.Context, key, contentType string) (PresignedRequest, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	res, err := c.presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		if c.uploadTTL > 0 {
			opts.Expires = c.uploadTTL
		}
	})
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("presign put object: %w", err)
	}

	return PresignedRequest{
		URL:     res.URL,
		Method:  res.Method,
		Headers: res.SignedHeader,
		Expires: expiresAt(c.uploadTTL),
	}, nil
}

func (c *Client) PresignGetObject(ctx context.Context, key string) (PresignedRequest, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	res, err := c.presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		if c.downloadTTL > 0 {
			opts.Expires = c.downloadTTL
		}
	})
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("presign get object: %w", err)
	}

	return PresignedRequest{
		URL:     res.URL,
		Method:  res.Method,
		Headers: res.SignedHeader,
		Expires: expiresAt(c.downloadTTL),
	}, nil
}

func (c *Client) HeadObject(ctx context.Context, key string) error {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("head object: %w", err)
	}
	return nil
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func expiresAt(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

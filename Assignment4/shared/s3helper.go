package shared

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the AWS S3 client with helper methods
type S3Client struct {
	Client *s3.Client
	Region string
}

// NewS3Client creates a new S3 client using default AWS credentials
func NewS3Client(region string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &S3Client{Client: client, Region: region}, nil
}

// ParseS3URL parses an S3 URL (s3://bucket/key) into bucket and key
func ParseS3URL(s3URL string) (bucket, key string, err error) {
	// Handle s3:// style URLs
	if strings.HasPrefix(s3URL, "s3://") {
		parsed, err := url.Parse(s3URL)
		if err != nil {
			return "", "", fmt.Errorf("invalid S3 URL: %w", err)
		}
		bucket = parsed.Host
		key = strings.TrimPrefix(parsed.Path, "/")
		return bucket, key, nil
	}

	// Handle https://bucket.s3.region.amazonaws.com/key style URLs
	if strings.Contains(s3URL, "s3") && strings.Contains(s3URL, "amazonaws.com") {
		parsed, err := url.Parse(s3URL)
		if err != nil {
			return "", "", fmt.Errorf("invalid S3 URL: %w", err)
		}
		hostParts := strings.Split(parsed.Host, ".")
		bucket = hostParts[0]
		key = strings.TrimPrefix(parsed.Path, "/")
		return bucket, key, nil
	}

	return "", "", fmt.Errorf("unrecognized S3 URL format: %s", s3URL)
}

// BuildS3URL constructs an S3 URL from bucket and key
func BuildS3URL(bucket, key string) string {
	return fmt.Sprintf("s3://%s/%s", bucket, key)
}

// ReadFromS3 downloads an object from S3 and returns its content as a string
func (sc *S3Client) ReadFromS3(ctx context.Context, s3URL string) (string, error) {
	bucket, key, err := ParseS3URL(s3URL)
	if err != nil {
		return "", err
	}

	output, err := sc.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get object from S3 (%s): %w", s3URL, err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return string(data), nil
}

// WriteToS3 uploads content to S3 at the specified bucket/key
func (sc *S3Client) WriteToS3(ctx context.Context, bucket, key, content, contentType string) (string, error) {
	_, err := sc.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to put object to S3: %w", err)
	}

	return BuildS3URL(bucket, key), nil
}
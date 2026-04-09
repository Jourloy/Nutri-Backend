package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/jourloy/somivyn/internal/lib"
)

const (
	FolderAI     = "ai"
	FolderBlog   = "blog"
	FolderRecipe = "recipe"
)

type Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	UseSSL     bool
}

type Service interface {
	EnsureFolder(ctx context.Context, folder string) error
	Upload(ctx context.Context, folder, key string, body []byte, contentType string) (string, error)
	BuildPublicURL(folder, key string) string
}

type s3API interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type service struct {
	client        s3API
	bucketName    string
	publicBaseURL string
	ensured       sync.Map
}

func NewS3Service(cfg Config) (Service, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("s3 endpoint is required")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" {
		return nil, fmt.Errorf("s3 access key is required")
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("s3 secret key is required")
	}
	if strings.TrimSpace(cfg.BucketName) == "" {
		return nil, fmt.Errorf("s3 bucket name is required")
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = "us-east-1"
	}

	baseURL, err := NormalizeBaseURL(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, err
	}

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               baseURL,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awsconfig.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load s3 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return newService(client, cfg.BucketName, baseURL)
}

func NewS3ServiceFromConfig() (Service, error) {
	return NewS3Service(Config{
		Endpoint:   lib.Config.S3Endpoint,
		AccessKey:  lib.Config.S3AccessKey,
		SecretKey:  lib.Config.S3SecretKey,
		BucketName: lib.Config.S3BucketName,
		Region:     lib.Config.S3Region,
		UseSSL:     lib.Config.S3UseSSL,
	})
}

func NormalizeBaseURL(endpoint string, useSSL bool) (string, error) {
	base := strings.TrimSpace(endpoint)
	if base == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		base = scheme + "://" + base
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("failed to parse s3 endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid s3 endpoint: %q", endpoint)
	}

	parsed.Path = "/" + strings.Trim(parsed.Path, "/")
	if parsed.Path == "/" {
		parsed.Path = ""
	}

	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func newService(client s3API, bucketName, publicBaseURL string) (Service, error) {
	if client == nil {
		return nil, fmt.Errorf("s3 client is required")
	}
	if strings.TrimSpace(bucketName) == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if strings.TrimSpace(publicBaseURL) == "" {
		return nil, fmt.Errorf("public base url is required")
	}

	return &service{
		client:        client,
		bucketName:    bucketName,
		publicBaseURL: strings.TrimSuffix(publicBaseURL, "/"),
	}, nil
}

func (s *service) EnsureFolder(ctx context.Context, folder string) error {
	folder = normalizeFolder(folder)
	if folder == "" {
		return fmt.Errorf("folder is required")
	}

	if _, ok := s.ensured.Load(folder); ok {
		return nil
	}

	prefix := folder + "/"
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to check folder %q: %w", folder, err)
	}

	if len(out.Contents) == 0 {
		_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucketName),
			Key:         aws.String(prefix),
			Body:        bytes.NewReader(nil),
			ContentType: aws.String("application/x-directory"),
		})
		if err != nil {
			return fmt.Errorf("failed to create folder marker %q: %w", folder, err)
		}
	}

	s.ensured.Store(folder, struct{}{})
	return nil
}

func (s *service) Upload(ctx context.Context, folder, key string, body []byte, contentType string) (string, error) {
	if err := s.EnsureFolder(ctx, folder); err != nil {
		return "", err
	}

	objectKey, err := joinObjectKey(folder, key)
	if err != nil {
		return "", err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object %q: %w", objectKey, err)
	}

	return buildPublicURL(s.publicBaseURL, s.bucketName, objectKey), nil
}

func (s *service) BuildPublicURL(folder, key string) string {
	objectKey, err := joinObjectKey(folder, key)
	if err != nil {
		return ""
	}
	return buildPublicURL(s.publicBaseURL, s.bucketName, objectKey)
}

func buildPublicURL(baseURL, bucketName, objectKey string) string {
	parsed, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return ""
	}

	segments := make([]string, 0, 3)
	if trimmed := strings.Trim(parsed.Path, "/"); trimmed != "" {
		segments = append(segments, trimmed)
	}
	segments = append(segments, strings.Trim(bucketName, "/"))
	segments = append(segments, strings.Trim(objectKey, "/"))
	parsed.Path = "/" + path.Join(segments...)

	return parsed.String()
}

func normalizeFolder(folder string) string {
	return strings.Trim(strings.TrimSpace(folder), "/")
}

func joinObjectKey(folder, key string) (string, error) {
	folder = normalizeFolder(folder)
	if folder == "" {
		return "", fmt.Errorf("folder is required")
	}

	trimmedKey := strings.Trim(strings.TrimSpace(key), "/")
	if trimmedKey == "" {
		return "", fmt.Errorf("key is required")
	}

	return path.Join(folder, trimmedKey), nil
}

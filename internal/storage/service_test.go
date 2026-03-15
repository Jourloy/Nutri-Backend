package storage

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3Client struct {
	listOutputs []*s3.ListObjectsV2Output
	listCalls   int
	putInputs   []*s3.PutObjectInput
}

func (f *fakeS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	index := f.listCalls
	f.listCalls++
	if index < len(f.listOutputs) && f.listOutputs[index] != nil {
		return f.listOutputs[index], nil
	}
	return &s3.ListObjectsV2Output{}, nil
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putInputs = append(f.putInputs, params)
	return &s3.PutObjectOutput{}, nil
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		useSSL   bool
		want     string
	}{
		{
			name:     "keeps explicit scheme",
			endpoint: "https://cdn.example.com/",
			useSSL:   false,
			want:     "https://cdn.example.com",
		},
		{
			name:     "adds https when ssl enabled",
			endpoint: "cdn.example.com",
			useSSL:   true,
			want:     "https://cdn.example.com",
		},
		{
			name:     "adds http when ssl disabled",
			endpoint: "localhost:9000",
			useSSL:   false,
			want:     "http://localhost:9000",
		},
		{
			name:     "preserves base path",
			endpoint: "https://cdn.example.com/storage/",
			useSSL:   true,
			want:     "https://cdn.example.com/storage",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeBaseURL(tt.endpoint, tt.useSSL)
			if err != nil {
				t.Fatalf("NormalizeBaseURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPublicURL(t *testing.T) {
	t.Parallel()

	svc, err := newService(&fakeS3Client{}, "nutri-images", "https://cdn.example.com/storage")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	got := svc.BuildPublicURL(FolderBlog, "2026/03/file.png")
	want := "https://cdn.example.com/storage/nutri-images/blog/2026/03/file.png"
	if got != want {
		t.Fatalf("BuildPublicURL() = %q, want %q", got, want)
	}
}

func TestEnsureFolderCreatesMarkerOnlyOnce(t *testing.T) {
	t.Parallel()

	client := &fakeS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{}},
	}
	svcRaw, err := newService(client, "nutri-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	svc := svcRaw.(*service)

	if err := svc.EnsureFolder(context.Background(), FolderAI); err != nil {
		t.Fatalf("EnsureFolder() error = %v", err)
	}
	if err := svc.EnsureFolder(context.Background(), FolderAI); err != nil {
		t.Fatalf("EnsureFolder() second call error = %v", err)
	}

	if client.listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", client.listCalls)
	}
	if len(client.putInputs) != 1 {
		t.Fatalf("expected 1 marker put, got %d", len(client.putInputs))
	}
	if got := aws.ToString(client.putInputs[0].Key); got != "ai/" {
		t.Fatalf("marker key = %q, want %q", got, "ai/")
	}
	if got := aws.ToString(client.putInputs[0].ContentType); got != "application/x-directory" {
		t.Fatalf("marker contentType = %q, want %q", got, "application/x-directory")
	}
}

func TestEnsureFolderSkipsMarkerWhenPrefixExists(t *testing.T) {
	t.Parallel()

	client := &fakeS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []types.Object{{Key: aws.String("recipe/existing.jpg")}},
		}},
	}
	svc, err := newService(client, "nutri-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	if err := svc.EnsureFolder(context.Background(), FolderRecipe); err != nil {
		t.Fatalf("EnsureFolder() error = %v", err)
	}
	if len(client.putInputs) != 0 {
		t.Fatalf("expected no marker creation, got %d put calls", len(client.putInputs))
	}
}

func TestUploadReturnsPathStyleURL(t *testing.T) {
	t.Parallel()

	client := &fakeS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{}},
	}
	svc, err := newService(client, "nutri-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	got, err := svc.Upload(context.Background(), FolderBlog, "2026/03/file.jpg", []byte("abc"), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if got != "https://cdn.example.com/nutri-images/blog/2026/03/file.jpg" {
		t.Fatalf("Upload() url = %q", got)
	}
	if len(client.putInputs) != 2 {
		t.Fatalf("expected marker + object upload, got %d puts", len(client.putInputs))
	}
	if got := aws.ToString(client.putInputs[1].Key); got != "blog/2026/03/file.jpg" {
		t.Fatalf("object key = %q, want %q", got, "blog/2026/03/file.jpg")
	}
}

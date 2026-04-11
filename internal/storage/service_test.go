package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type fakeS3Client struct {
	listOutputs []*s3.ListObjectsV2Output
	listCalls   int
	putInputs   []*s3.PutObjectInput
	getOutput   *s3.GetObjectOutput
	getErr      error
	getInput    *s3.GetObjectInput
	headOutput  *s3.HeadObjectOutput
	headErr     error
	headInput   *s3.HeadObjectInput
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

func (f *fakeS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.getInput = params
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getOutput != nil {
		return f.getOutput, nil
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	f.headInput = params
	if f.headErr != nil {
		return nil, f.headErr
	}
	if f.headOutput != nil {
		return f.headOutput, nil
	}
	return &s3.HeadObjectOutput{}, nil
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

	svc, err := newService(&fakeS3Client{}, "somivyn-images", "https://cdn.example.com/storage")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	got := svc.BuildPublicURL(FolderBlog, "2026/03/file.png")
	want := "https://cdn.example.com/storage/somivyn-images/blog/2026/03/file.png"
	if got != want {
		t.Fatalf("BuildPublicURL() = %q, want %q", got, want)
	}
}

func TestResolvePublicBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		endpoint      string
		publicBaseURL string
		useSSL        bool
		want          string
	}{
		{
			name:          "uses explicit public base url",
			endpoint:      "http://internal.example.com:9000",
			publicBaseURL: "https://cdn.example.com/storage",
			useSSL:        false,
			want:          "https://cdn.example.com/storage",
		},
		{
			name:          "falls back to endpoint when public base url missing",
			endpoint:      "minio.example.com",
			publicBaseURL: "",
			useSSL:        true,
			want:          "https://minio.example.com",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolvePublicBaseURL(tt.endpoint, tt.publicBaseURL, tt.useSSL)
			if err != nil {
				t.Fatalf("resolvePublicBaseURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePublicBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureFolderCreatesMarkerOnlyOnce(t *testing.T) {
	t.Parallel()

	client := &fakeS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{}},
	}
	svcRaw, err := newService(client, "somivyn-images", "https://cdn.example.com")
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
	svc, err := newService(client, "somivyn-images", "https://cdn.example.com")
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
	svc, err := newService(client, "somivyn-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	got, err := svc.Upload(context.Background(), FolderBlog, "2026/03/file.jpg", []byte("abc"), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if got != "https://cdn.example.com/somivyn-images/blog/2026/03/file.jpg" {
		t.Fatalf("Upload() url = %q", got)
	}
	if len(client.putInputs) != 2 {
		t.Fatalf("expected marker + object upload, got %d puts", len(client.putInputs))
	}
	if got := aws.ToString(client.putInputs[1].Key); got != "blog/2026/03/file.jpg" {
		t.Fatalf("object key = %q, want %q", got, "blog/2026/03/file.jpg")
	}
}

func TestGetObjectReturnsBodyAndMetadata(t *testing.T) {
	t.Parallel()

	modifiedAt := time.Now().UTC().Truncate(time.Second)
	client := &fakeS3Client{
		getOutput: &s3.GetObjectOutput{
			Body:          io.NopCloser(bytes.NewReader([]byte("image-bytes"))),
			ContentType:   aws.String("image/png"),
			ContentLength: aws.Int64(11),
			ETag:          aws.String(`"etag-1"`),
			LastModified:  &modifiedAt,
		},
	}
	svc, err := newService(client, "somivyn-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	object, err := svc.GetObject(context.Background(), FolderBlog, "2026/03/file.png")
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	defer object.Body.Close()

	body, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(body) != "image-bytes" {
		t.Fatalf("body = %q", string(body))
	}
	if object.ContentType != "image/png" {
		t.Fatalf("contentType = %q", object.ContentType)
	}
	if object.ContentLength != 11 {
		t.Fatalf("contentLength = %d", object.ContentLength)
	}
	if object.ETag != `"etag-1"` {
		t.Fatalf("etag = %q", object.ETag)
	}
	if object.LastModified == nil || !object.LastModified.Equal(modifiedAt) {
		t.Fatalf("lastModified = %#v", object.LastModified)
	}
	if got := aws.ToString(client.getInput.Key); got != "blog/2026/03/file.png" {
		t.Fatalf("object key = %q", got)
	}
}

func TestHeadObjectReturnsMetadata(t *testing.T) {
	t.Parallel()

	modifiedAt := time.Now().UTC().Truncate(time.Second)
	client := &fakeS3Client{
		headOutput: &s3.HeadObjectOutput{
			ContentType:   aws.String("image/webp"),
			ContentLength: aws.Int64(27),
			ETag:          aws.String(`"etag-2"`),
			LastModified:  &modifiedAt,
		},
	}
	svc, err := newService(client, "somivyn-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	object, err := svc.HeadObject(context.Background(), FolderBlog, "2026/03/file.webp")
	if err != nil {
		t.Fatalf("HeadObject() error = %v", err)
	}

	if object.ContentType != "image/webp" {
		t.Fatalf("contentType = %q", object.ContentType)
	}
	if object.ContentLength != 27 {
		t.Fatalf("contentLength = %d", object.ContentLength)
	}
	if object.ETag != `"etag-2"` {
		t.Fatalf("etag = %q", object.ETag)
	}
	if object.LastModified == nil || !object.LastModified.Equal(modifiedAt) {
		t.Fatalf("lastModified = %#v", object.LastModified)
	}
	if got := aws.ToString(client.headInput.Key); got != "blog/2026/03/file.webp" {
		t.Fatalf("object key = %q", got)
	}
}

func TestGetObjectMapsNotFound(t *testing.T) {
	t.Parallel()

	client := &fakeS3Client{
		getErr: &types.NoSuchKey{},
	}
	svc, err := newService(client, "somivyn-images", "https://cdn.example.com")
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	_, err = svc.GetObject(context.Background(), FolderBlog, "missing.png")
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

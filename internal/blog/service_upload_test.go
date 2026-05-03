package blog

import (
	"context"
	"regexp"
	"testing"

	"github.com/jourloy/nutri02/internal/storage"
)

type fakeStorage struct {
	folder      string
	key         string
	body        []byte
	contentType string
	url         string
}

func (f *fakeStorage) EnsureFolder(ctx context.Context, folder string) error { return nil }

func (f *fakeStorage) Upload(ctx context.Context, folder, key string, body []byte, contentType string) (string, error) {
	f.folder = folder
	f.key = key
	f.body = append([]byte(nil), body...)
	f.contentType = contentType
	return f.url, nil
}

func (f *fakeStorage) BuildPublicURL(folder, key string) string { return "" }
func (f *fakeStorage) GetObject(ctx context.Context, folder, key string) (*storage.ObjectReader, error) {
	return nil, nil
}
func (f *fakeStorage) HeadObject(ctx context.Context, folder, key string) (*storage.ObjectInfo, error) {
	return nil, nil
}

func TestUploadImage_UsesBlogFolderAndContentType(t *testing.T) {
	t.Parallel()

	storageFake := &fakeStorage{url: "https://cdn.example.com/somivyn-images/blog/2026/03/cover.png"}
	urlCanonicalizer, err := storage.NewBlogImageURLCanonicalizer(storage.Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizer() error = %v", err)
	}
	imageURLMapper, err := newBlogImageURLMapper(urlCanonicalizer, "https://api.example.com")
	if err != nil {
		t.Fatalf("newBlogImageURLMapper() error = %v", err)
	}
	svc := &service{storage: storageFake, imageURLMapper: imageURLMapper}

	payload := []byte("png-bytes")
	gotURL, err := svc.UploadImage(context.Background(), payload, "cover.png")
	if err != nil {
		t.Fatalf("UploadImage() error = %v", err)
	}

	if matched := regexp.MustCompile(`^https://api\.example\.com/api/v1/blog/images/\d{4}/\d{2}/[0-9a-f-]+\.png$`).MatchString(gotURL); !matched {
		t.Fatalf("unexpected proxy url %q", gotURL)
	}
	if storageFake.folder != storage.FolderBlog {
		t.Fatalf("folder = %q, want %q", storageFake.folder, storage.FolderBlog)
	}
	if matched := regexp.MustCompile(`^\d{4}/\d{2}/[0-9a-f-]+\.png$`).MatchString(storageFake.key); !matched {
		t.Fatalf("unexpected key %q", storageFake.key)
	}
	if storageFake.contentType != "image/png" {
		t.Fatalf("contentType = %q, want %q", storageFake.contentType, "image/png")
	}
	if string(storageFake.body) != string(payload) {
		t.Fatalf("payload changed during upload")
	}
}

package blog

import (
	"context"
	"regexp"
	"testing"

	"github.com/jourloy/somivyn/internal/storage"
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

func TestUploadImage_UsesBlogFolderAndContentType(t *testing.T) {
	t.Parallel()

	storageFake := &fakeStorage{url: "https://cdn.example.com/somivyn-images/blog/2026/03/cover.png"}
	svc := &service{storage: storageFake}

	payload := []byte("png-bytes")
	gotURL, err := svc.UploadImage(context.Background(), payload, "cover.png")
	if err != nil {
		t.Fatalf("UploadImage() error = %v", err)
	}

	if gotURL != storageFake.url {
		t.Fatalf("url = %q, want %q", gotURL, storageFake.url)
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

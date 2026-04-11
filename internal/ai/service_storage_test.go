package ai

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
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
func (f *fakeStorage) GetObject(ctx context.Context, folder, key string) (*storage.ObjectReader, error) {
	return nil, nil
}
func (f *fakeStorage) HeadObject(ctx context.Context, folder, key string) (*storage.ObjectInfo, error) {
	return nil, nil
}

func TestProcessAndUploadImage_UsesAIFolderAndJPEG(t *testing.T) {
	t.Parallel()

	var input bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&input, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	storageFake := &fakeStorage{url: "https://cdn.example.com/somivyn-images/ai/user-1/file.jpg"}
	svc := &service{storage: storageFake}

	gotURL, base64Image, err := svc.processAndUploadImage(context.Background(), "user-1", input.Bytes())
	if err != nil {
		t.Fatalf("processAndUploadImage() error = %v", err)
	}

	if gotURL != storageFake.url {
		t.Fatalf("url = %q, want %q", gotURL, storageFake.url)
	}
	if storageFake.folder != storage.FolderAI {
		t.Fatalf("folder = %q, want %q", storageFake.folder, storage.FolderAI)
	}
	if matched := regexp.MustCompile(`^user-1/[0-9a-f-]+_[0-9]+\.jpg$`).MatchString(storageFake.key); !matched {
		t.Fatalf("unexpected key %q", storageFake.key)
	}
	if storageFake.contentType != "image/jpeg" {
		t.Fatalf("contentType = %q, want %q", storageFake.contentType, "image/jpeg")
	}
	if len(storageFake.body) < 2 || storageFake.body[0] != 0xff || storageFake.body[1] != 0xd8 {
		t.Fatalf("expected jpeg payload, got %d bytes", len(storageFake.body))
	}
	if base64Image == "" {
		t.Fatalf("expected non-empty base64 payload")
	}
}

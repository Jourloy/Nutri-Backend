package blog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type imageServiceStub struct {
	Service
	image    *ImageObject
	err      error
	lastKey  string
	lastHead bool
}

func (s *imageServiceStub) GetImage(ctx context.Context, key string, headOnly bool) (*ImageObject, error) {
	s.lastKey = key
	s.lastHead = headOnly
	if s.err != nil {
		return nil, s.err
	}
	return s.image, nil
}

func TestGetImageStreamsPublicImage(t *testing.T) {
	t.Parallel()

	modifiedAt := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	stub := &imageServiceStub{
		image: &ImageObject{
			Body:          io.NopCloser(strings.NewReader("png-bytes")),
			ContentType:   "image/png",
			ContentLength: 9,
			ETag:          `"etag-1"`,
			LastModified:  &modifiedAt,
		},
	}

	router := chi.NewRouter()
	controller := &Controller{service: stub}
	controller.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/blog/images/2026/03/cover.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if stub.lastKey != "2026/03/cover.png" {
		t.Fatalf("key = %q", stub.lastKey)
	}
	if stub.lastHead {
		t.Fatalf("expected GET request")
	}
	if rr.Body.String() != "png-bytes" {
		t.Fatalf("body = %q", rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("cache-control = %q", got)
	}
	if got := rr.Header().Get("ETag"); got != `"etag-1"` {
		t.Fatalf("etag = %q", got)
	}
	if got := rr.Header().Get("Last-Modified"); got != modifiedAt.Format(http.TimeFormat) {
		t.Fatalf("last-modified = %q", got)
	}
}

func TestGetImageHeadReturnsHeadersOnly(t *testing.T) {
	t.Parallel()

	stub := &imageServiceStub{
		image: &ImageObject{
			ContentType:   "image/webp",
			ContentLength: 27,
		},
	}

	router := chi.NewRouter()
	controller := &Controller{service: stub}
	controller.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodHead, "/blog/images/2026/03/cover.webp", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if !stub.lastHead {
		t.Fatalf("expected HEAD request")
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "image/webp" {
		t.Fatalf("content-type = %q", got)
	}
}

func TestGetImageNotFound(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	controller := &Controller{service: &imageServiceStub{err: ErrImageNotFound}}
	controller.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/blog/images/2026/03/missing.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestGetImageInvalidKey(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	controller := &Controller{service: &imageServiceStub{err: ErrInvalidImageKey}}
	controller.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/blog/images/%2e%2e/cover.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetImageStorageFailure(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	controller := &Controller{service: &imageServiceStub{err: errors.New("boom")}}
	controller.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/blog/images/2026/03/cover.png", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d", http.StatusBadGateway, rr.Code)
	}
}

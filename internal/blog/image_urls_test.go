package blog

import (
	"testing"

	"github.com/jourloy/somivyn/internal/storage"
)

func newTestBlogImageURLMapper(t *testing.T) *blogImageURLMapper {
	t.Helper()

	canonicalizer, err := storage.NewBlogImageURLCanonicalizer(storage.Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizer() error = %v", err)
	}

	mapper, err := newBlogImageURLMapper(canonicalizer, "https://api.example.com")
	if err != nil {
		t.Fatalf("newBlogImageURLMapper() error = %v", err)
	}

	return mapper
}

func TestBlogImageURLMapperRewriteURLForStorage(t *testing.T) {
	t.Parallel()

	mapper := newTestBlogImageURLMapper(t)

	tests := []struct {
		name    string
		input   string
		want    string
		changed bool
	}{
		{
			name:    "proxy url becomes canonical s3",
			input:   "https://api.example.com/api/v1/blog/images/2026/03/cover.png?size=lg#hero",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png?size=lg#hero",
			changed: true,
		},
		{
			name:    "api alias becomes canonical s3",
			input:   "https://api.somivyn.com/somivyn-blog-images/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "canonical s3 stays canonical",
			input:   "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: false,
		},
		{
			name:    "external image is untouched",
			input:   "https://example.com/assets/cover.png",
			want:    "https://example.com/assets/cover.png",
			changed: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, changed := mapper.RewriteURLForStorage(tt.input)
			if changed != tt.changed {
				t.Fatalf("RewriteURLForStorage() changed = %v, want %v", changed, tt.changed)
			}
			if got != tt.want {
				t.Fatalf("RewriteURLForStorage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlogImageURLMapperRewriteURLForDelivery(t *testing.T) {
	t.Parallel()

	mapper := newTestBlogImageURLMapper(t)

	tests := []struct {
		name    string
		input   string
		want    string
		changed bool
	}{
		{
			name:    "relative alias becomes proxy",
			input:   "/somivyn-blog-images/2026/03/cover.png?size=lg#hero",
			want:    "https://api.example.com/api/v1/blog/images/2026/03/cover.png?size=lg#hero",
			changed: true,
		},
		{
			name:    "legacy minio alias becomes proxy",
			input:   "https://minio.jourloy.com/somivyn-blog-images/2026/03/cover.png",
			want:    "https://api.example.com/api/v1/blog/images/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "canonical s3 becomes proxy",
			input:   "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			want:    "https://api.example.com/api/v1/blog/images/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "proxy url is normalized but unchanged",
			input:   "https://api.example.com/api/v1/blog/images/2026/03/cover.png",
			want:    "https://api.example.com/api/v1/blog/images/2026/03/cover.png",
			changed: false,
		},
		{
			name:    "external image is untouched",
			input:   "https://example.com/assets/cover.png",
			want:    "https://example.com/assets/cover.png",
			changed: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, changed := mapper.RewriteURLForDelivery(tt.input)
			if changed != tt.changed {
				t.Fatalf("RewriteURLForDelivery() changed = %v, want %v", changed, tt.changed)
			}
			if got != tt.want {
				t.Fatalf("RewriteURLForDelivery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlogImageURLMapperRewriteTextForDelivery(t *testing.T) {
	t.Parallel()

	mapper := newTestBlogImageURLMapper(t)

	input := `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png" /></p>`
	got, changed := mapper.RewriteTextForDelivery(input)
	if !changed {
		t.Fatalf("expected html to change")
	}

	want := `<p><img src="https://api.example.com/api/v1/blog/images/2026/03/cover.png" /></p>`
	if got != want {
		t.Fatalf("RewriteTextForDelivery() = %q, want %q", got, want)
	}
}

func TestNormalizeBlogImageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "keeps nested key",
			input: "2026/03/cover.png",
			want:  "2026/03/cover.png",
		},
		{
			name:    "rejects parent traversal",
			input:   "../cover.png",
			wantErr: true,
		},
		{
			name:    "rejects empty key",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBlogImageKey(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeBlogImageKey() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeBlogImageKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

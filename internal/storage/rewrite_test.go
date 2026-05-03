package storage

import "testing"

func TestLegacyURLRewriterRewriteURL(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "somivyn-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	got, changed := rewriter.RewriteURL("http://legacy.example.com:9000/nutri-ai-images/user-1/file.jpg")
	if !changed {
		t.Fatalf("expected URL to change")
	}

	want := "https://cdn.example.com/somivyn-images/ai/user-1/file.jpg"
	if got != want {
		t.Fatalf("RewriteURL() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterRewriteText(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "somivyn-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	input := `![cover](http://legacy.example.com:9000/nutri-blog-images/2026/03/cover.png)`
	got, changed := rewriter.RewriteText(input)
	if !changed {
		t.Fatalf("expected text to change")
	}

	want := `![cover](https://cdn.example.com/somivyn-images/blog/2026/03/cover.png)`
	if got != want {
		t.Fatalf("RewriteText() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterRewriteMetadata(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "somivyn-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	input := `{"violationId":1,"userId":"u1","imageUrl":"http://legacy.example.com:9000/nutri-ai-images/u1/file.jpg"}`
	got, changed := rewriter.RewriteMetadata(input)
	if !changed {
		t.Fatalf("expected metadata to change")
	}

	want := `{"imageUrl":"https://cdn.example.com/somivyn-images/ai/u1/file.jpg","userId":"u1","violationId":1}`
	if got != want {
		t.Fatalf("RewriteMetadata() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterIgnoresExternalAndMigratedURLs(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "somivyn-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	cases := []string{
		"https://example.com/other/file.jpg",
		"https://cdn.example.com/somivyn-images/blog/2026/03/cover.png",
	}

	for _, input := range cases {
		got, changed := rewriter.RewriteURL(input)
		if changed {
			t.Fatalf("expected no change for %q, got %q", input, got)
		}
		if got != input {
			t.Fatalf("expected original value for %q, got %q", input, got)
		}
	}
}

func TestBlogImageURLCanonicalizerRewriteURL(t *testing.T) {
	t.Parallel()

	rewriter, err := NewBlogImageURLCanonicalizer(Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizer() error = %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		changed bool
	}{
		{
			name:    "relative somivyn alias",
			input:   "/nutri02-blog-images/2026/03/cover.png?size=lg#hero",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png?size=lg#hero",
			changed: true,
		},
		{
			name:    "api domain alias",
			input:   "https://api.nutri02.com/nutri02-blog-images/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "front domain alias",
			input:   "https://nutri02.com/nutri02-blog-images/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "legacy minio alias",
			input:   "https://minio.jourloy.com/nutri02-blog-images/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "internal direct s3 path",
			input:   "https://internal.example.com/somivyn-images/blog/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "legacy s3 bucket path",
			input:   "https://s3.nutri02.com/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/blog/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "legacy twc bucket path",
			input:   "https://s3.twcstorage.ru/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/blog/2026/03/cover.png",
			want:    "https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png",
			changed: true,
		},
		{
			name:    "already canonical",
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

			got, changed := rewriter.RewriteURL(tt.input)
			if changed != tt.changed {
				t.Fatalf("RewriteURL() changed = %v, want %v", changed, tt.changed)
			}
			if got != tt.want {
				t.Fatalf("RewriteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlogImageURLCanonicalizerRewriteText(t *testing.T) {
	t.Parallel()

	rewriter, err := NewBlogImageURLCanonicalizer(Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizer() error = %v", err)
	}

	input := `<p><img src="/nutri02-blog-images/2026/03/cover.png" /></p>`
	got, changed := rewriter.RewriteText(input)
	if !changed {
		t.Fatalf("expected text to change")
	}

	want := `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/cover.png" /></p>`
	if got != want {
		t.Fatalf("RewriteText() = %q, want %q", got, want)
	}
}

func TestRecipeImageURLCanonicalizerRewriteURL(t *testing.T) {
	t.Parallel()

	rewriter, err := NewRecipeImageURLCanonicalizer(Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewRecipeImageURLCanonicalizer() error = %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		changed bool
	}{
		{
			name:    "relative somivyn alias",
			input:   "/nutri02-recipe-images/2026/03/step.webp?size=lg#hero",
			want:    "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp?size=lg#hero",
			changed: true,
		},
		{
			name:    "api domain alias",
			input:   "https://api.nutri02.com/nutri02-recipe-images/2026/03/step.webp",
			want:    "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp",
			changed: true,
		},
		{
			name:    "minio alias",
			input:   "https://minio.jourloy.com/nutri02-recipe-images/2026/03/step.webp",
			want:    "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp",
			changed: true,
		},
		{
			name:    "legacy s3 bucket path",
			input:   "https://s3.nutri02.com/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/recipe/2026/03/step.webp",
			want:    "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp",
			changed: true,
		},
		{
			name:    "already canonical",
			input:   "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp",
			want:    "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp",
			changed: false,
		},
		{
			name:    "external image is untouched",
			input:   "https://example.com/assets/step.webp",
			want:    "https://example.com/assets/step.webp",
			changed: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, changed := rewriter.RewriteURL(tt.input)
			if changed != tt.changed {
				t.Fatalf("RewriteURL() changed = %v, want %v", changed, tt.changed)
			}
			if got != tt.want {
				t.Fatalf("RewriteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

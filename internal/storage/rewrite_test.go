package storage

import "testing"

func TestLegacyURLRewriterRewriteURL(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "nutri-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	got, changed := rewriter.RewriteURL("http://legacy.example.com:9000/nutri-ai-images/user-1/file.jpg")
	if !changed {
		t.Fatalf("expected URL to change")
	}

	want := "https://cdn.example.com/nutri-images/ai/user-1/file.jpg"
	if got != want {
		t.Fatalf("RewriteURL() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterRewriteText(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "nutri-images",
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

	want := `![cover](https://cdn.example.com/nutri-images/blog/2026/03/cover.png)`
	if got != want {
		t.Fatalf("RewriteText() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterRewriteMetadata(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "nutri-images",
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

	want := `{"imageUrl":"https://cdn.example.com/nutri-images/ai/u1/file.jpg","userId":"u1","violationId":1}`
	if got != want {
		t.Fatalf("RewriteMetadata() = %q, want %q", got, want)
	}
}

func TestLegacyURLRewriterIgnoresExternalAndMigratedURLs(t *testing.T) {
	t.Parallel()

	rewriter, err := NewLegacyURLRewriter("http://legacy.example.com:9000", false, Config{
		Endpoint:   "https://cdn.example.com",
		BucketName: "nutri-images",
		UseSSL:     true,
	})
	if err != nil {
		t.Fatalf("NewLegacyURLRewriter() error = %v", err)
	}

	cases := []string{
		"https://example.com/other/file.jpg",
		"https://cdn.example.com/nutri-images/blog/2026/03/cover.png",
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

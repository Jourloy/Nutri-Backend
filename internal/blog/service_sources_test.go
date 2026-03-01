package blog

import "testing"

func TestNormalizeSourceURLs(t *testing.T) {
	in := []string{
		" https://Example.com/path ",
		"https://example.com/path",
		"http://example.com/another",
		"ftp://example.com/no",
		"example.com/no-scheme",
		"",
	}

	out := normalizeSourceURLs(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 urls, got %#v", out)
	}
	if out[0] != "https://example.com/path" {
		t.Fatalf("unexpected first url: %q", out[0])
	}
	if out[1] != "http://example.com/another" {
		t.Fatalf("unexpected second url: %q", out[1])
	}
}

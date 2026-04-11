package blog

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/jourloy/somivyn/internal/storage"
)

const blogImageProxyPathPrefix = "/api/v1/blog/images"

type blogImageURLMapper struct {
	canonicalizer *storage.BlogImageURLCanonicalizer
	proxyBaseURL  *url.URL
	textPattern   *regexp.Regexp
}

func newBlogImageURLMapper(
	canonicalizer *storage.BlogImageURLCanonicalizer,
	backendBaseURL string,
) (*blogImageURLMapper, error) {
	if canonicalizer == nil {
		return nil, fmt.Errorf("blog url canonicalizer is required")
	}

	baseURL := strings.TrimSpace(backendBaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("backend base url is required")
	}

	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	parsed, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse backend base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid backend base url: %q", backendBaseURL)
	}

	parsed.Path = "/" + strings.Trim(parsed.Path, "/")
	if parsed.Path == "/" {
		parsed.Path = ""
	}

	return &blogImageURLMapper{
		canonicalizer: canonicalizer,
		proxyBaseURL:  parsed,
		textPattern: regexp.MustCompile(
			`https?://[^\s"'<>)]+|/(?:somivyn|nutri)-blog-images/[^\s"'<>)]+`,
		),
	}, nil
}

func (m *blogImageURLMapper) RewriteURLForStorage(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}

	if key, rawQuery, fragment, ok := m.extractProxyObjectKey(trimmed); ok {
		rewritten := m.buildCanonicalURL(key, rawQuery, fragment)
		return rewritten, rewritten != raw
	}

	return m.canonicalizer.RewriteURL(trimmed)
}

func (m *blogImageURLMapper) RewriteTextForStorage(raw string) (string, bool) {
	if raw == "" {
		return raw, false
	}

	changed := false
	rewritten := m.textPattern.ReplaceAllStringFunc(raw, func(match string) string {
		next, ok := m.RewriteURLForStorage(match)
		if ok {
			changed = true
			return next
		}
		return match
	})

	return rewritten, changed
}

func (m *blogImageURLMapper) RewriteURLForDelivery(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}

	if key, rawQuery, fragment, ok := m.extractProxyObjectKey(trimmed); ok {
		rewritten := m.BuildProxyURL(key, rawQuery, fragment)
		return rewritten, rewritten != raw
	}

	key, ok := m.canonicalizer.ExtractObjectKey(trimmed)
	if !ok {
		return raw, false
	}

	rawQuery := ""
	fragment := ""
	if parsed, err := url.Parse(trimmed); err == nil {
		rawQuery = parsed.RawQuery
		fragment = parsed.Fragment
	}

	rewritten := m.BuildProxyURL(key, rawQuery, fragment)
	return rewritten, rewritten != raw
}

func (m *blogImageURLMapper) RewriteTextForDelivery(raw string) (string, bool) {
	if raw == "" {
		return raw, false
	}

	changed := false
	rewritten := m.textPattern.ReplaceAllStringFunc(raw, func(match string) string {
		next, ok := m.RewriteURLForDelivery(match)
		if ok {
			changed = true
			return next
		}
		return match
	})

	return rewritten, changed
}

func (m *blogImageURLMapper) BuildProxyURL(key, rawQuery, fragment string) string {
	trimmedKey := strings.Trim(strings.TrimSpace(key), "/")
	if trimmedKey == "" {
		return ""
	}

	segments := make([]string, 0, 6)
	if basePath := strings.Trim(m.proxyBaseURL.Path, "/"); basePath != "" {
		segments = append(segments, basePath)
	}
	segments = append(segments, strings.Trim(blogImageProxyPathPrefix, "/"))
	segments = append(segments, trimmedKey)

	rewritten := *m.proxyBaseURL
	rewritten.Path = "/" + path.Join(segments...)
	rewritten.RawQuery = rawQuery
	rewritten.Fragment = fragment
	return rewritten.String()
}

func (m *blogImageURLMapper) extractProxyObjectKey(raw string) (string, string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", "", false
	}
	if !parsed.IsAbs() {
		return "", "", "", false
	}
	if !strings.EqualFold(parsed.Scheme, m.proxyBaseURL.Scheme) || !strings.EqualFold(parsed.Host, m.proxyBaseURL.Host) {
		return "", "", "", false
	}

	candidates := []string{normalizeProxyPath(parsed.Path)}
	if trimmed, ok := trimProxyBasePath(parsed.Path, m.proxyBaseURL.Path); ok {
		normalized := normalizeProxyPath(trimmed)
		if normalized != candidates[0] {
			candidates = append(candidates, normalized)
		}
	}

	prefix := normalizeProxyPath(blogImageProxyPathPrefix) + "/"
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate, prefix) {
			continue
		}

		key := strings.Trim(strings.TrimPrefix(candidate, prefix), "/")
		if key == "" {
			return "", "", "", false
		}
		return key, parsed.RawQuery, parsed.Fragment, true
	}

	return "", "", "", false
}

func (m *blogImageURLMapper) buildCanonicalURL(key, rawQuery, fragment string) string {
	rewritten := m.canonicalizer.BuildCanonicalURL(key)
	if rewritten == "" {
		return ""
	}

	parsed, err := url.Parse(rewritten)
	if err != nil {
		return ""
	}
	parsed.RawQuery = rawQuery
	parsed.Fragment = fragment
	return parsed.String()
}

func normalizeProxyPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	return "/" + strings.Trim(trimmed, "/")
}

func trimProxyBasePath(fullPath, basePath string) (string, bool) {
	cleanFull := normalizeProxyPath(fullPath)
	cleanBase := normalizeProxyPath(basePath)
	if cleanBase == "/" {
		return cleanFull, true
	}
	if cleanFull == cleanBase {
		return "/", true
	}
	prefix := cleanBase + "/"
	if !strings.HasPrefix(cleanFull, prefix) {
		return "", false
	}
	return strings.TrimPrefix(cleanFull, cleanBase), true
}

package storage

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var legacyFolderMap = map[string]string{
	"nutri-ai-images":     FolderAI,
	"nutri-blog-images":   FolderBlog,
	"nutri-recipe-images": FolderRecipe,
}

type LegacyURLRewriter struct {
	legacyBaseURL *url.URL
	targetBaseURL string
	targetBucket  string
	textPattern   *regexp.Regexp
}

func NewLegacyURLRewriter(legacyEndpoint string, legacyUseSSL bool, target Config) (*LegacyURLRewriter, error) {
	legacyBase, err := NormalizeBaseURL(legacyEndpoint, legacyUseSSL)
	if err != nil {
		return nil, err
	}

	targetBase, err := NormalizeBaseURL(target.Endpoint, target.UseSSL)
	if err != nil {
		return nil, err
	}

	parsedLegacy, err := url.Parse(legacyBase)
	if err != nil {
		return nil, err
	}

	buckets := make([]string, 0, len(legacyFolderMap))
	for bucket := range legacyFolderMap {
		buckets = append(buckets, regexp.QuoteMeta(bucket))
	}

	return &LegacyURLRewriter{
		legacyBaseURL: parsedLegacy,
		targetBaseURL: targetBase,
		targetBucket:  target.BucketName,
		textPattern: regexp.MustCompile(
			regexp.QuoteMeta(strings.TrimSuffix(legacyBase, "/")) + `/(` + strings.Join(buckets, "|") + `)/[^\s"'<>)]+`,
		),
	}, nil
}

func (r *LegacyURLRewriter) RewriteURL(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return raw, false
	}
	if !sameOrigin(parsed, r.legacyBaseURL) {
		return raw, false
	}

	relativePath, ok := trimBasePath(parsed.Path, r.legacyBaseURL.Path)
	if !ok {
		return raw, false
	}

	parts := strings.Split(strings.Trim(relativePath, "/"), "/")
	if len(parts) < 2 {
		return raw, false
	}

	folder, ok := legacyFolderMap[parts[0]]
	if !ok {
		return raw, false
	}

	rewritten := buildPublicURL(r.targetBaseURL, r.targetBucket, path.Join(folder, path.Join(parts[1:]...)))
	if rewritten == "" {
		return raw, false
	}

	rewrittenURL, err := url.Parse(rewritten)
	if err != nil {
		return raw, false
	}
	rewrittenURL.RawQuery = parsed.RawQuery
	rewrittenURL.Fragment = parsed.Fragment

	return rewrittenURL.String(), rewrittenURL.String() != raw
}

func (r *LegacyURLRewriter) RewriteText(raw string) (string, bool) {
	if raw == "" {
		return raw, false
	}

	changed := false
	rewritten := r.textPattern.ReplaceAllStringFunc(raw, func(match string) string {
		next, ok := r.RewriteURL(match)
		if ok {
			changed = true
			return next
		}
		return match
	})

	return rewritten, changed
}

func (r *LegacyURLRewriter) RewriteMetadata(raw string) (string, bool) {
	if strings.TrimSpace(raw) == "" {
		return raw, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw, false
	}

	imageURL, ok := payload["imageUrl"].(string)
	if !ok {
		return raw, false
	}

	rewritten, changed := r.RewriteURL(imageURL)
	if !changed {
		return raw, false
	}
	payload["imageUrl"] = rewritten

	encoded, err := json.Marshal(payload)
	if err != nil {
		return raw, false
	}

	return string(encoded), true
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func trimBasePath(fullPath, basePath string) (string, bool) {
	cleanFull := "/" + strings.Trim(fullPath, "/")
	cleanBase := "/" + strings.Trim(basePath, "/")
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

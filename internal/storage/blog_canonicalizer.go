package storage

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const blogCanonicalizerSampleKey = "__blog_image_canonicalizer__"

var (
	blogImageAliasPrefixes = []string{
		"/nutri02-blog-images/",
		"/somivyn-blog-images/",
		"/nutri-blog-images/",
	}
	legacyBlogBucketNames = []string{
		"cd83329f-b1dd-42b6-afac-9af67c6c8cc1",
	}
	defaultKnownImageHosts = []string{
		"api.nutri.jourloy.com",
		"api.nutri02.jourloy.com",
		"api.nutri02.com",
		"api.somivyn.com",
		"api.somivyn.jourloy.com",
		"api-somivyn.jourloy.com",
		"api-nutri.jourloy.com",
		"minio-somivyn.jourloy.com",
		"minio.jourloy.com",
		"nutri.jourloy.com",
		"nutri02.jourloy.com",
		"s3.nutri02.com",
		"s3.somivyn.com",
		"s3.twcstorage.ru",
		"nutri02.com",
		"somivyn.com",
		"somivyn.jourloy.com",
		"72.56.69.80:9000",
		"82.29.179.36:9000",
	}
)

type BlogImageURLCanonicalizer struct {
	targetBaseURL   *url.URL
	endpointBaseURL *url.URL
	targetBucket    string
	knownHosts      map[string]struct{}
	textPattern     *regexp.Regexp
}

func NewBlogImageURLCanonicalizer(cfg Config) (*BlogImageURLCanonicalizer, error) {
	if strings.TrimSpace(cfg.BucketName) == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	publicBaseURL, err := resolvePublicBaseURL(cfg.Endpoint, cfg.PublicBaseURL, cfg.UseSSL)
	if err != nil {
		return nil, err
	}

	endpointBaseURL := ""
	if strings.TrimSpace(cfg.Endpoint) != "" {
		endpointBaseURL, err = NormalizeBaseURL(cfg.Endpoint, cfg.UseSSL)
		if err != nil {
			return nil, err
		}
	}

	return newBlogImageURLCanonicalizer(publicBaseURL, endpointBaseURL, cfg.BucketName)
}

func NewBlogImageURLCanonicalizerFromService(svc Service) (*BlogImageURLCanonicalizer, error) {
	if svc == nil {
		return nil, fmt.Errorf("storage service is required")
	}

	sampleURL := strings.TrimSpace(svc.BuildPublicURL(FolderBlog, blogCanonicalizerSampleKey))
	if sampleURL == "" {
		return nil, fmt.Errorf("storage service returned empty public url")
	}

	parsed, err := url.Parse(sampleURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sample public url: %w", err)
	}

	segments := splitPathSegments(parsed.Path)
	if len(segments) < 3 {
		return nil, fmt.Errorf("unexpected sample public url path: %q", parsed.Path)
	}
	if segments[len(segments)-2] != FolderBlog || segments[len(segments)-1] != blogCanonicalizerSampleKey {
		return nil, fmt.Errorf("unexpected sample public url path: %q", parsed.Path)
	}

	bucketName := segments[len(segments)-3]
	baseSegments := segments[:len(segments)-3]
	if len(baseSegments) == 0 {
		parsed.Path = ""
	} else {
		parsed.Path = "/" + path.Join(baseSegments...)
	}

	return newBlogImageURLCanonicalizer(parsed.String(), "", bucketName)
}

func newBlogImageURLCanonicalizer(publicBaseURL, endpointBaseURL, bucketName string) (*BlogImageURLCanonicalizer, error) {
	parsedPublicBaseURL, err := url.Parse(strings.TrimSuffix(publicBaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public base url: %w", err)
	}
	if parsedPublicBaseURL.Scheme == "" || parsedPublicBaseURL.Host == "" {
		return nil, fmt.Errorf("invalid public base url: %q", publicBaseURL)
	}

	var parsedEndpointBaseURL *url.URL
	if strings.TrimSpace(endpointBaseURL) != "" {
		parsedEndpointBaseURL, err = url.Parse(strings.TrimSuffix(endpointBaseURL, "/"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse endpoint base url: %w", err)
		}
		if parsedEndpointBaseURL.Scheme == "" || parsedEndpointBaseURL.Host == "" {
			return nil, fmt.Errorf("invalid endpoint base url: %q", endpointBaseURL)
		}
	}

	knownHosts := make(map[string]struct{}, len(defaultKnownImageHosts)+2)
	for _, host := range defaultKnownImageHosts {
		if normalizedHost := strings.ToLower(strings.TrimSpace(host)); normalizedHost != "" {
			knownHosts[normalizedHost] = struct{}{}
		}
	}
	knownHosts[strings.ToLower(parsedPublicBaseURL.Host)] = struct{}{}
	if parsedEndpointBaseURL != nil {
		knownHosts[strings.ToLower(parsedEndpointBaseURL.Host)] = struct{}{}
	}

	return &BlogImageURLCanonicalizer{
		targetBaseURL:   parsedPublicBaseURL,
		endpointBaseURL: parsedEndpointBaseURL,
		targetBucket:    strings.Trim(strings.TrimSpace(bucketName), "/"),
		knownHosts:      knownHosts,
		textPattern: regexp.MustCompile(
			`https?://[^\s"'<>)]+|/(?:nutri02|somivyn|nutri)-blog-images/[^\s"'<>)]+`,
		),
	}, nil
}

func (r *BlogImageURLCanonicalizer) RewriteURL(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return raw, false
	}
	if !parsed.IsAbs() {
		key, ok := r.extractBlogImageKeyFromPath(parsed.Path)
		if !ok {
			return raw, false
		}

		rewritten := r.buildURL(key, parsed.RawQuery, parsed.Fragment)
		return rewritten, rewritten != raw
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return raw, false
	}
	if !r.isKnownHost(parsed.Host) {
		return raw, false
	}

	key, ok := r.extractBlogImageKeyFromPath(parsed.Path)
	if !ok {
		return raw, false
	}

	rewritten := r.buildURL(key, parsed.RawQuery, parsed.Fragment)
	return rewritten, rewritten != raw
}

func (r *BlogImageURLCanonicalizer) RewriteText(raw string) (string, bool) {
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

func (r *BlogImageURLCanonicalizer) BuildCanonicalURL(key string) string {
	trimmedKey := strings.Trim(strings.TrimSpace(key), "/")
	if trimmedKey == "" {
		return ""
	}
	return r.buildURL(trimmedKey, "", "")
}

func (r *BlogImageURLCanonicalizer) ExtractObjectKey(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if !parsed.IsAbs() {
		return r.extractBlogImageKeyFromPath(parsed.Path)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if !r.isKnownHost(parsed.Host) {
		return "", false
	}

	return r.extractBlogImageKeyFromPath(parsed.Path)
}

func (r *BlogImageURLCanonicalizer) buildURL(key, rawQuery, fragment string) string {
	rewritten := buildPublicURL(r.targetBaseURL.String(), r.targetBucket, path.Join(FolderBlog, key))
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

func (r *BlogImageURLCanonicalizer) extractBlogImageKeyFromPath(rawPath string) (string, bool) {
	if key, ok := extractFolderImageKey(rawPath, blogImageAliasPrefixes); ok {
		return key, true
	}

	for _, candidatePath := range r.pathCandidates(rawPath) {
		if key, ok := extractFolderImageKeyFromBucketPath(candidatePath, r.targetBucket, FolderBlog); ok {
			return key, true
		}

		for _, legacyBucket := range legacyBlogBucketNames {
			if key, ok := extractFolderImageKeyFromBucketPath(candidatePath, legacyBucket, FolderBlog); ok {
				return key, true
			}
		}
	}

	return "", false
}

func (r *BlogImageURLCanonicalizer) pathCandidates(rawPath string) []string {
	candidates := []string{normalizePath(rawPath)}
	for _, basePath := range []string{r.targetBaseURL.Path, urlPathOrEmpty(r.endpointBaseURL)} {
		if basePath == "" {
			continue
		}
		trimmed, ok := trimBasePath(rawPath, basePath)
		if !ok {
			continue
		}
		normalized := normalizePath(trimmed)
		alreadyIncluded := false
		for _, candidate := range candidates {
			if candidate == normalized {
				alreadyIncluded = true
				break
			}
		}
		if !alreadyIncluded {
			candidates = append(candidates, normalized)
		}
	}
	return candidates
}

func (r *BlogImageURLCanonicalizer) isKnownHost(host string) bool {
	_, ok := r.knownHosts[strings.ToLower(strings.TrimSpace(host))]
	return ok
}

func extractFolderImageKey(raw string, aliasPrefixes []string) (string, bool) {
	normalized := normalizePath(raw)
	for _, prefix := range aliasPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			key := strings.Trim(strings.TrimPrefix(normalized, prefix), "/")
			if key != "" {
				return key, true
			}
		}
	}

	return "", false
}

func extractFolderImageKeyFromBucketPath(rawPath, bucketName, folder string) (string, bool) {
	normalized := normalizePath(rawPath)
	prefix := "/" + strings.Trim(bucketName, "/") + "/" + folder + "/"
	if !strings.HasPrefix(normalized, prefix) {
		return "", false
	}

	key := strings.Trim(strings.TrimPrefix(normalized, prefix), "/")
	if key == "" {
		return "", false
	}
	return key, true
}

func normalizePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	return "/" + strings.Trim(trimmed, "/")
}

func splitPathSegments(rawPath string) []string {
	normalized := strings.Trim(normalizePath(rawPath), "/")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "/")
}

func urlPathOrEmpty(value *url.URL) string {
	if value == nil {
		return ""
	}
	return value.Path
}

package storage

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

const recipeCanonicalizerSampleKey = "__recipe_image_canonicalizer__"

var (
	recipeImageAliasPrefixes = []string{
		"/somivyn-recipe-images/",
		"/nutri-recipe-images/",
	}
	legacyRecipeBucketNames = []string{
		"cd83329f-b1dd-42b6-afac-9af67c6c8cc1",
	}
)

type RecipeImageURLCanonicalizer struct {
	targetBaseURL   *url.URL
	endpointBaseURL *url.URL
	targetBucket    string
	knownHosts      map[string]struct{}
}

func NewRecipeImageURLCanonicalizer(cfg Config) (*RecipeImageURLCanonicalizer, error) {
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

	return newRecipeImageURLCanonicalizer(publicBaseURL, endpointBaseURL, cfg.BucketName)
}

func newRecipeImageURLCanonicalizer(publicBaseURL, endpointBaseURL, bucketName string) (*RecipeImageURLCanonicalizer, error) {
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

	return &RecipeImageURLCanonicalizer{
		targetBaseURL:   parsedPublicBaseURL,
		endpointBaseURL: parsedEndpointBaseURL,
		targetBucket:    strings.Trim(strings.TrimSpace(bucketName), "/"),
		knownHosts:      knownHosts,
	}, nil
}

func (r *RecipeImageURLCanonicalizer) RewriteURL(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return raw, false
	}
	if !parsed.IsAbs() {
		key, ok := r.extractRecipeImageKeyFromPath(parsed.Path)
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

	key, ok := r.extractRecipeImageKeyFromPath(parsed.Path)
	if !ok {
		return raw, false
	}

	rewritten := r.buildURL(key, parsed.RawQuery, parsed.Fragment)
	return rewritten, rewritten != raw
}

func (r *RecipeImageURLCanonicalizer) buildURL(key, rawQuery, fragment string) string {
	rewritten := buildPublicURL(r.targetBaseURL.String(), r.targetBucket, path.Join(FolderRecipe, key))
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

func (r *RecipeImageURLCanonicalizer) extractRecipeImageKeyFromPath(rawPath string) (string, bool) {
	if key, ok := extractFolderImageKey(rawPath, recipeImageAliasPrefixes); ok {
		return key, true
	}

	for _, candidatePath := range r.pathCandidates(rawPath) {
		if key, ok := extractFolderImageKeyFromBucketPath(candidatePath, r.targetBucket, FolderRecipe); ok {
			return key, true
		}

		for _, legacyBucket := range legacyRecipeBucketNames {
			if key, ok := extractFolderImageKeyFromBucketPath(candidatePath, legacyBucket, FolderRecipe); ok {
				return key, true
			}
		}
	}

	return "", false
}

func (r *RecipeImageURLCanonicalizer) pathCandidates(rawPath string) []string {
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

func (r *RecipeImageURLCanonicalizer) isKnownHost(host string) bool {
	_, ok := r.knownHosts[strings.ToLower(strings.TrimSpace(host))]
	return ok
}

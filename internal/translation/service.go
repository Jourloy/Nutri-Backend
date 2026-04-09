package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/redis/go-redis/v9"

	"github.com/jourloy/somivyn/internal/cache"
)

type NamespaceMap map[string]map[string]string

type Payload struct {
	Namespaces NamespaceMap `json:"namespaces"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

type Service struct {
	repo  Repository
	cache *redis.Client
}

func NewService() Service {
	return Service{
		repo:  NewRepository(),
		cache: cache.Client(),
	}
}

const cacheTTL = 15 * time.Minute

func (s Service) cacheKey(locale string) string {
	return fmt.Sprintf("translations:%s", strings.ToLower(locale))
}

func (s Service) GetByLocale(ctx context.Context, locale string) (Payload, error) {
	locale = strings.ToLower(locale)
	if locale == "" {
		locale = "ru"
	}

	if fromCache := s.readFromCache(ctx, locale); fromCache != nil {
		return *fromCache, nil
	}

	items, err := s.repo.GetByLocale(ctx, locale)
	if err != nil {
		return Payload{}, err
	}

	result := make(NamespaceMap)
	var lastUpdated time.Time
	for _, item := range items {
		ns := item.Namespace
		if _, ok := result[ns]; !ok {
			result[ns] = make(map[string]string)
		}
		result[ns][item.Key] = item.Value
		if item.UpdatedAt.After(lastUpdated) {
			lastUpdated = item.UpdatedAt
		}
	}

	payload := Payload{
		Namespaces: result,
		UpdatedAt:  lastUpdated,
	}

	_ = s.writeToCache(ctx, locale, payload)

	return payload, nil
}

func (s Service) Upsert(ctx context.Context, req UpsertRequest) (*Translation, error) {
	req.Locale = strings.ToLower(req.Locale)
	item, err := s.repo.Upsert(ctx, req)
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx, req.Locale)
	return item, nil
}

func (s Service) Delete(ctx context.Context, req DeleteRequest) error {
	req.Locale = strings.ToLower(req.Locale)
	if err := s.repo.SoftDelete(ctx, req); err != nil {
		return err
	}
	s.invalidate(ctx, req.Locale)
	return nil
}

func (s Service) readFromCache(ctx context.Context, locale string) *Payload {
	if s.cache == nil {
		return nil
	}
	data, err := s.cache.Get(ctx, s.cacheKey(locale)).Result()
	if err != nil {
		if err != redis.Nil {
			log.Error("redis: failed to read translations", "error", err)
		}
		return nil
	}
	var res Payload
	if err := json.Unmarshal([]byte(data), &res); err != nil {
		log.Error("redis: failed to unmarshal translations", "error", err)
		return nil
	}
	return &res
}

func (s Service) writeToCache(ctx context.Context, locale string, payload Payload) error {
	if s.cache == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, s.cacheKey(locale), data, cacheTTL).Err()
}

func (s Service) invalidate(ctx context.Context, locale string) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Del(ctx, s.cacheKey(locale)).Err(); err != nil && err != redis.Nil {
		log.Error("redis: failed to invalidate translations", "error", err)
	}
}

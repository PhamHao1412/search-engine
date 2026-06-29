package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"search-service/internal/entity"
	"search-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new ProductCache implementation using Redis
func NewRedisCache(client *redis.Client) service.ProductCache {
	return &redisCache{client: client}
}

func (c *redisCache) CacheProduct(ctx context.Context, tenantID, productID string, data map[string]interface{}) error {
	redisKey := fmt.Sprintf("tenant:%s:product:%s", tenantID, productID)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, redisKey, string(val), 24*time.Hour).Err()
}

func (c *redisCache) GetCachedSearch(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, string, bool, error) {
	redisKey := fmt.Sprintf("search:%s:%s:%d:%d", tenantID, query, page, pageSize)
	val, err := c.client.Get(ctx, redisKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, 0, "", false, nil
	} else if err != nil {
		return nil, 0, "", false, err
	}

	var cached struct {
		Total       int                      `json:"total"`
		Products    []map[string]interface{} `json:"products"`
		SearchLogID string                   `json:"search_log_id"`
	}
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, 0, "", false, err
	}
	return cached.Products, cached.Total, cached.SearchLogID, true, nil
}

func (c *redisCache) CacheSearch(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int, searchLogID string) error {
	redisKey := fmt.Sprintf("search:%s:%s:%d:%d", tenantID, query, page, pageSize)
	cached := struct {
		Total       int                      `json:"total"`
		Products    []map[string]interface{} `json:"products"`
		SearchLogID string                   `json:"search_log_id"`
	}{
		Total:       total,
		Products:    data,
		SearchLogID: searchLogID,
	}
	val, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, redisKey, string(val), 10*time.Minute).Err()
}

func (c *redisCache) GetCachedSuggestions(ctx context.Context, tenantID, query string) ([]entity.Suggestion, bool, error) {
	redisKey := fmt.Sprintf("suggest:%s:%s", tenantID, query)
	val, err := c.client.Get(ctx, redisKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	var suggestions []entity.Suggestion
	if err := json.Unmarshal([]byte(val), &suggestions); err != nil {
		return nil, false, err
	}
	return suggestions, true, nil
}

func (c *redisCache) CacheSuggestions(ctx context.Context, tenantID, query string, suggestions []entity.Suggestion) error {
	redisKey := fmt.Sprintf("suggest:%s:%s", tenantID, query)
	val, err := json.Marshal(suggestions)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, redisKey, string(val), 5*time.Minute).Err()
}

func (c *redisCache) GetCachedSpellcheck(ctx context.Context, tenantID, typoWord string) (string, bool, error) {
	redisKey := fmt.Sprintf("tenant:%s:spellcheck:%s", tenantID, typoWord)
	val, err := c.client.Get(ctx, redisKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *redisCache) CacheSpellcheck(ctx context.Context, tenantID, typoWord, correctWord string) error {
	redisKey := fmt.Sprintf("tenant:%s:spellcheck:%s", tenantID, typoWord)
	ttl := 24 * time.Hour
	if correctWord == "-" {
		ttl = 5 * time.Minute
	}
	return c.client.Set(ctx, redisKey, correctWord, ttl).Err()
}

func (c *redisCache) DeleteSpellcheckCache(ctx context.Context, tenantID, typoWord string) error {
	redisKey := fmt.Sprintf("tenant:%s:spellcheck:%s", tenantID, typoWord)
	return c.client.Del(ctx, redisKey).Err()
}

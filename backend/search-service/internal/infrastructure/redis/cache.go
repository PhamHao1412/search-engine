package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"search-service/internal/service"
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

func (c *redisCache) GetCachedSearch(ctx context.Context, tenantID, query string, page, pageSize int) ([]map[string]interface{}, int, bool, error) {
	redisKey := fmt.Sprintf("search:%s:%s:%d:%d", tenantID, query, page, pageSize)
	val, err := c.client.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		return nil, 0, false, nil
	} else if err != nil {
		return nil, 0, false, err
	}

	var cached struct {
		Total    int                      `json:"total"`
		Products []map[string]interface{} `json:"products"`
	}
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, 0, false, err
	}
	return cached.Products, cached.Total, true, nil
}

func (c *redisCache) CacheSearch(ctx context.Context, tenantID, query string, page, pageSize int, data []map[string]interface{}, total int) error {
	redisKey := fmt.Sprintf("search:%s:%s:%d:%d", tenantID, query, page, pageSize)
	cached := struct {
		Total    int                      `json:"total"`
		Products []map[string]interface{} `json:"products"`
	}{
		Total:    total,
		Products: data,
	}
	val, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, redisKey, string(val), 10*time.Minute).Err()
}

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redirectTTL    = 1 * time.Hour
	uniqueVisitTTL = 24 * time.Hour
)

type LinkCache struct {
	rdb *redis.Client
}

func NewLinkCache(rdb *redis.Client) *LinkCache {
	return &LinkCache{rdb: rdb}
}

// SetLink caches the shortCode → originalURL mapping.
func (c *LinkCache) SetLink(ctx context.Context, domainName, shortCode, originalURL string) error {
	return c.rdb.Set(ctx, redirectKey(domainName, shortCode), originalURL, redirectTTL).Err()
}

// GetLink retrieves the cached originalURL for shortCode.
// Returns ("", nil) if the key does not exist (cache miss).
func (c *LinkCache) GetLink(ctx context.Context, domainName, shortCode string) (string, error) {
	val, err := c.rdb.Get(ctx, redirectKey(domainName, shortCode)).Result()
	if err == redis.Nil {
		return "", nil // cache miss — not an error
	}
	return val, err
}

// DeleteLink removes a shortCode from the cache (e.g. on update/delete).
func (c *LinkCache) DeleteLink(ctx context.Context, domainName, shortCode string) error {
	return c.rdb.Del(ctx, redirectKey(domainName, shortCode)).Err()
}

// IsUniqueVisitor uses SETNX to determine if this IP has visited this link
// in the past 24 hours. Returns true the first time (unique visit).
func (c *LinkCache) IsUniqueVisitor(ctx context.Context, domainName, shortCode, ipHash string) (bool, error) {
	key := fmt.Sprintf("uniq:%s:%s:%s", domainName, shortCode, ipHash)
	ok, err := c.rdb.SetNX(ctx, key, "1", uniqueVisitTTL).Result()
	return ok, err
}

// IncrRateLimit increments and returns the request counter for the given
// identifier within the current 1-minute window.
func (c *LinkCache) IncrRateLimit(ctx context.Context, identifier string) (int64, error) {
	key := fmt.Sprintf("rate:%s:%d", identifier, time.Now().Unix()/60)
	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 70*time.Second) // slightly longer than 60s window
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func redirectKey(domainName, shortCode string) string {
	if domainName == "" {
		return "redirect::" + shortCode
	}
	return "redirect:" + domainName + ":" + shortCode
}


package repositories

import (
	"context"
	"errors"
	"time"

	"alert/db"

	"github.com/go-redis/redis/v8"
)

type rateLimitEntity struct {
	resource *db.Resource
}

type IRateLimit interface {
	Increment(key string, window time.Duration) (int64, error)
	// Get returns 0 for a key that does not exist and an error when Redis fails.
	Get(key string) (int64, error)
	Reset(key string) error
}

func NewRateLimitEntity(resource *db.Resource) IRateLimit {
	return &rateLimitEntity{resource: resource}
}

func (entity *rateLimitEntity) Increment(key string, window time.Duration) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := entity.resource.RdDb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		entity.resource.RdDb.Expire(ctx, key, window)
	}
	return count, nil
}

func (entity *rateLimitEntity) Get(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count, err := entity.resource.RdDb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (entity *rateLimitEntity) Reset(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return entity.resource.RdDb.Del(ctx, key).Err()
}

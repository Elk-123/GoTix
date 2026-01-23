package storage

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/atomic_lock.lua
var lockScript string

type Provider interface {
	Lock(ctx context.Context, eventID string, offset int64, userID string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, eventID string, offset int64) error
}

type redisProvider struct {
	client *redis.Client
	prefix string
}

func NewRedisProvider(client *redis.Client, prefix string) Provider {
	return &redisProvider{
		client: client,
		prefix: prefix,
	}
}

func (r *redisProvider) Lock(ctx context.Context, eventID string, offset int64, userID string, ttl time.Duration) (bool, error) {
	bitmapKey := fmt.Sprintf("%s:%s:seats", r.prefix, eventID)
	lockDetailKey := fmt.Sprintf("%s:%s:lock:%d", r.prefix, eventID, offset)

	res, err := r.client.Eval(ctx, lockScript,
		[]string{bitmapKey, lockDetailKey},
		offset, userID, int(ttl.Seconds()),
	).Result()

	if err != nil {
		return false, err
	}

	return res.(int64) == 1, nil
}

func (r *redisProvider) Unlock(ctx context.Context, eventID string, offset int64) error {
	bitmapKey := fmt.Sprintf("%s:%s:seats", r.prefix, eventID)
	lockDetailKey := fmt.Sprintf("%s:%s:lock:%d", r.prefix, eventID, offset)

	pipe := r.client.Pipeline()
	pipe.SetBit(ctx, bitmapKey, offset, 0)
	pipe.Del(ctx, lockDetailKey)
	_, err := pipe.Exec(ctx)
	return err
}
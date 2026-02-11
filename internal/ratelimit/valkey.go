package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type ValkeyLimiter struct {
	client *redis.Client
	prefix string
}

func NewValkeyLimiter(addr, password string) *ValkeyLimiter {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})

	return &ValkeyLimiter{
		client: client,
		prefix: "rl:",
	}
}

func (l *ValkeyLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if l == nil || l.client == nil {
		return true, nil
	}

	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()
	redisKey := l.prefix + key
	member := fmt.Sprintf("%d-%s", now, randomSuffix())

	pipe := l.client.Pipeline()
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now), Member: member})
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, redisKey)
	pipe.Expire(ctx, redisKey, window+time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return countCmd.Val() <= int64(limit), nil
}

func randomSuffix() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf)
}

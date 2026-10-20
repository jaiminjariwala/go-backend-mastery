package main

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

var errCacheMiss = errors.New("cache miss")

type infrastructure interface {
	Get(context.Context, string) (string, error)
	Put(context.Context, string, string) error
	Allow(context.Context, string) (bool, time.Duration, error)
}
type redisInfra struct{ client *redis.Client }

func (b *redisInfra) Get(ctx context.Context, code string) (string, error) {
	value, err := b.client.Get(ctx, "shortener:v1:link:"+code).Result()
	if errors.Is(err, redis.Nil) {
		return "", errCacheMiss
	}
	return value, err
}
func (b *redisInfra) Put(ctx context.Context, code, url string) error {
	return b.client.Set(ctx, "shortener:v1:link:"+code, url, 60*time.Second).Err()
}

// One atomic script updates the shared counter and its expiry together.
// The window starts on the first request, not at a wall-clock boundary.
var limitScript = redis.NewScript(`
local n=redis.call('INCR',KEYS[1])
if n==1 then redis.call('PEXPIRE',KEYS[1],10000) end
return {n,redis.call('PTTL',KEYS[1])}
`)

func (b *redisInfra) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	values, err := limitScript.Run(ctx, b.client, []string{"shortener:v1:limit:" + key}).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	return values[0] <= 20, time.Duration(values[1]) * time.Millisecond, nil
}

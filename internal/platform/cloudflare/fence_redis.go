package cloudflare

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisFence stores the single-writer fence in the existing Redis database.
type RedisFence struct {
	Client *redis.Client
}

func (r RedisFence) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if r.Client == nil {
		return false, errors.New("cloudflare writer fence redis is missing")
	}
	return r.Client.SetNX(ctx, key, value, ttl).Result()
}

func (r RedisFence) Get(ctx context.Context, key string) (string, bool, error) {
	if r.Client == nil {
		return "", false, errors.New("cloudflare writer fence redis is missing")
	}
	value, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (r RedisFence) CompareAndPExpire(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if r.Client == nil {
		return false, errors.New("cloudflare writer fence redis is missing")
	}
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`
	result, err := r.Client.Eval(ctx, script, []string{key}, value, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (r RedisFence) CompareAndDelete(ctx context.Context, key, value string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("cloudflare writer fence redis is missing")
	}
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	result, err := r.Client.Eval(ctx, script, []string{key}, value).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

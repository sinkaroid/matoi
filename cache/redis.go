// Package cache provides a wrapper around Redis for caching responses.
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RDB is the global shared Redis client instance.
var RDB *redis.Client

// InitRedis initializes the shared Redis client and verifies connection.
func InitRedis(addr string) error {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		return err
	}

	RDB = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return RDB.Ping(ctx).Err()
}

// Get retrieves a key from Redis, unmarshaling it into dest. Returns true if found.
func Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := RDB.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Set serializes value to JSON and stores it in Redis with the given TTL.
func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return RDB.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key from Redis.
func Delete(ctx context.Context, key string) error {
	return RDB.Del(ctx, key).Err()
}

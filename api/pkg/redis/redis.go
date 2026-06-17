package redis

import (
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewClient() (*redis.Client, error) {
	rawURL := os.Getenv("REDIS_URL")
	if rawURL == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}
	opt, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	return redis.NewClient(opt), nil
}

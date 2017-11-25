package main

import (
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
)

var redisPool *redis.Pool

func initRedisPool() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	redisPool = &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 10 * 60 * time.Second,

		Dial: func() (redis.Conn, error) {
			return redis.DialURL(redisURL)
		},
	}
}

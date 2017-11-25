package main

import (
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	redisPool       *redis.Pool
	sharedRedisPool *redis.Pool
)

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
	conn := redisPool.Get()
	defer conn.Close()
	conn.Do("FLUSHALL")

	sharedRedisURL := os.Getenv("SHARED_REDIS_URL")
	if sharedRedisURL == "" {
		sharedRedisURL = "redis://localhost:6379"
	}
	sharedRedisPool = &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 10 * 60 * time.Second,

		Dial: func() (redis.Conn, error) {
			return redis.DialURL(sharedRedisURL)
		},
	}

	sharedconn := sharedRedisPool.Get()
	defer sharedconn.Close()
	sharedconn.Do("DEL", "host:member_count")
}

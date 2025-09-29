package utils

import (
	"github.com/redis/go-redis/v9"
)

// GetRedisClient converts interface{} to *redis.Client
func GetRedisClient(client interface{}) (*redis.Client, bool) {
	c, ok := client.(*redis.Client)
	return c, ok
}

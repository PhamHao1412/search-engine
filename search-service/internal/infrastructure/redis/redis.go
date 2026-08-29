package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(addr, pwd string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Redis cache.")
	return rdb, nil
}

package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepository struct {
	Redis *redis.Client
}

func NewTokenBlacklistRepository(redisClient *redis.Client) *TokenBlacklistRepository {
	return &TokenBlacklistRepository{Redis: redisClient}
}

func (r *TokenBlacklistRepository) Blacklist(tokenHash string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil //already expired, no need to store
	}
	ctx := context.Background()
	return r.Redis.Set(ctx, "blacklist:"+tokenHash, "1", ttl).Err()
}

func (r *TokenBlacklistRepository) IsBlacklisted(tokenHash string) (bool, error) {
	ctx := context.Background()
	_, err := r.Redis.Get(ctx, "blacklist:"+tokenHash).Result()
	if err == redis.Nil {
		return false, nil // not found, not blacklisted
	}
	if err != nil {
		return false, err // some other error
	}
	return true, nil // found, blacklisted
}

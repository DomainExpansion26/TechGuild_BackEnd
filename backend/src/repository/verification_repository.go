package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type VerificationRepository struct {
	Redis *redis.Client
}

func NewVerificationRepository(redisClient *redis.Client) *VerificationRepository {
	return &VerificationRepository{
		Redis: redisClient,
	}
}

func (r *VerificationRepository) SaveVerificationToken(userID string, token string) error {

	ctx := context.Background()

	return r.Redis.Set(
		ctx,
		"verify:"+token,
		userID,
		24*time.Hour,
	).Err()
}

func (r *VerificationRepository) GetVerificationToken(token string) (string, error) {

	ctx := context.Background()

	return r.Redis.Get(
		ctx,
		"verify:"+token,
	).Result()
}

func (r *VerificationRepository) DeleteVerificationToken(token string) error {

	ctx := context.Background()

	return r.Redis.Del(
		ctx,
		"verify:"+token,
	).Err()
}

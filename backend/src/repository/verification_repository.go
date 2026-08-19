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

	oldToken, err := r.Redis.Get(ctx, "verify:user:"+userID).Result()
	if err == nil && oldToken != "" {
		r.Redis.Del(ctx, "verify:"+oldToken)
	}

	// save new token

	if err := r.Redis.Set(ctx, "verify:"+token, userID, 24*time.Hour).Err(); err != nil {
		return err
	}

	return r.Redis.Set(
		ctx,
		"verify:user:"+userID,
		token,
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

func (r *VerificationRepository) SaveConsumedVerificationToken(token string, userID string) error {
	ctx := context.Background()

	return r.Redis.Set(
		ctx,
		"verify:consumed:"+token,
		userID,
		10*time.Minute,
	).Err()
}

func (r *VerificationRepository) GoConsumeVerificationToken(token string) (string, error) {
	ctx := context.Background()

	return r.Redis.Get(
		ctx,
		"verify:consumed:"+token,
	).Result()
}

func (r *VerificationRepository) DeleteVerificationToken(token string) error {

	ctx := context.Background()

	userID, err := r.Redis.Get(ctx, "verify:"+token).Result()
	if err == nil {
		r.Redis.Del(ctx, "verify:user:"+userID)
	}

	return r.Redis.Del(
		ctx,
		"verify:token:"+token,
	).Err()
}

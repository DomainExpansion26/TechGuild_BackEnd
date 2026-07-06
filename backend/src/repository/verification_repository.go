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

func (r *VerificationRepository) SaveOTP(email string, otp string) error {
	ctx := context.Background()
	return r.Redis.Set(ctx, "verify:"+email, otp, 10*time.Minute).Err()
}

func (r *VerificationRepository) GetOTP(email string) (string, error) {
	ctx := context.Background()
	return r.Redis.Get(ctx, "verify:"+email).Result()
}

func (r *VerificationRepository) DeleteOTP(email string) error {
	ctx := context.Background()
	return r.Redis.Del(ctx, "verify:"+email).Err()
}

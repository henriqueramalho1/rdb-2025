package repositories

import (
	"context"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
)

type HealthRepository struct {
	client *redis.Client
}

func NewHealthRepository(client *redis.Client) *HealthRepository {
	return &HealthRepository{
		client: client,
	}
}

func (r *HealthRepository) IsProcessorFailing(ctx context.Context, processor models.ProcessorType) bool {
	_, err := r.client.Get(ctx, string(processor)+":failing").Result()
	if err == redis.Nil {
		return false // If the key does not exist, we assume the processor is not failing
	}

	if err != nil {
		return false
	}

	return true
}

func (r *HealthRepository) SetProcessorStatus(ctx context.Context, processor models.ProcessorType, on bool) {
	if on {
		r.client.Del(ctx, string(processor)+":failing").Err()
		return
	}

	isFailingStr := "true"
	r.client.Set(ctx, string(processor)+":failing", isFailingStr, 0).Err()
}

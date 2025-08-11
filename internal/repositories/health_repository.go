package repositories

import (
	"context"
	"encoding/json"

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

func (r *HealthRepository) SetCoordinatorFlag() (bool, error) {
	success, err := r.client.SetNX(context.Background(), "coordinator", "true", 30).Result()
	if err != nil {
		return false, err
	}

	return success, nil
}

func (r *HealthRepository) GetProcessorStatus(processor models.ProcessorType) (models.ProcessorStatus, error) {
	data, err := r.client.Get(context.Background(), string(processor)).Result()
	if err != nil {
		return models.ProcessorStatus{}, err
	}

	var status models.ProcessorStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return models.ProcessorStatus{}, err
	}
	return status, nil
}

func (r *HealthRepository) SetProcessorStatus(processor models.ProcessorType, status models.ProcessorStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return r.client.Set(context.Background(), string(processor), data, 0).Err()
}

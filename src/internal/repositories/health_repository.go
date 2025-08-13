package repositories

import (
	"context"
	"strconv"

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
	isFailingStr, err := r.client.Get(context.Background(), string(processor)+":failing").Result()
	if err != nil && err != redis.Nil {
		return models.ProcessorStatus{}, err
	}

	status := models.ProcessorStatus{}
	switch isFailingStr {
	case "1":
		status.Failing = true
	default:
		status.Failing = false
	}

	responseTimeStr, err := r.client.Get(context.Background(), string(processor)+":response_time").Result()
	if err != nil && err != redis.Nil {
		return models.ProcessorStatus{}, err
	}

	if responseTimeStr == "" {
		status.MinResponseTime = 0
		return status, nil
	}

	status.MinResponseTime, err = strconv.Atoi(responseTimeStr)
	if err != nil {
		return models.ProcessorStatus{}, err
	}

	return status, nil
}

func (r *HealthRepository) SetProcessorStatus(processor models.ProcessorType, status models.ProcessorStatus) error {
	ctx := context.Background()
	err := r.client.Set(ctx, string(processor)+":failing", status.Failing, 0).Err()
	if err != nil {
		return err
	}

	err = r.client.Set(ctx, string(processor)+":response_time", status.MinResponseTime, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

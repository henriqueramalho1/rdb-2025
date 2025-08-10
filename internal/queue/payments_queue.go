package queue

import (
	"context"
	"encoding/json"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
)

const PaymentQueueName = "payments"

type PaymentsQueue struct {
	client *redis.Client
}

func NewPaymentsQueue(client *redis.Client) *PaymentsQueue {
	return &PaymentsQueue{client: client}
}

func (q *PaymentsQueue) Publish(ctx context.Context, msg *models.PaymentRequest) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return q.client.LPush(ctx, PaymentQueueName, data).Err()
}

func (q *PaymentsQueue) Consume(ctx context.Context) (*models.PaymentRequest, error) {
	result, err := q.client.BRPop(ctx, 0, PaymentQueueName).Result()
	if err != nil {
		return nil, err
	}

	var data models.PaymentRequest
	err = json.Unmarshal([]byte(result[1]), &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

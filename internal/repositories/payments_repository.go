package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
)

const PaymentQueueName = "payments"

type PaymentsRepository struct {
	r *redis.Client
}

func NewPaymentsRepository(r *redis.Client) *PaymentsRepository {
	return &PaymentsRepository{
		r: r,
	}
}

func (q *PaymentsRepository) Publish(ctx context.Context, data []byte) error {
	q.r.LPush(ctx, PaymentQueueName, data)
	return nil
}

func (q *PaymentsRepository) Consume(ctx context.Context) ([]byte, error) {
	result, err := q.r.BRPop(ctx, 1*time.Second, PaymentQueueName).Result()
	if err != nil {
		return nil, err
	}

	data := []byte(result[1])
	return data, nil
}

func (r *PaymentsRepository) StorePayment(ctx context.Context, processor models.ProcessorType, payment *models.PaymentRequest) error {

	paymentKey := "payment:" + payment.CorrelationId
	err := r.r.HSet(ctx, paymentKey, map[string]interface{}{
		"id":           payment.CorrelationId,
		"amount":       payment.Amount,
		"processed_at": payment.RequestedAt.UnixNano(),
		"processor":    string(processor),
	}).Err()
	if err != nil {
		return errors.New("failed to create payment: " + err.Error())
	}

	processorKey := "payments:" + string(processor)
	err = r.r.ZAdd(ctx, processorKey, redis.Z{
		Score:  float64(payment.RequestedAt.UnixNano()),
		Member: payment.CorrelationId,
	}).Err()
	if err != nil {
		return errors.New("failed to index payment: " + err.Error())
	}

	return nil
}

func (r *PaymentsRepository) GetPaymentsSummary(ctx context.Context, from, to time.Time) (*models.GlobalPaymentsSummary, error) {
	summary := &models.GlobalPaymentsSummary{}

	processors := []string{"default", "fallback"}

	for _, processor := range processors {
		processorKey := "payments:" + processor

		paymentIds, err := r.r.ZRangeByScore(ctx, processorKey, &redis.ZRangeBy{
			Min: fmt.Sprintf("%d", from.UnixNano()),
			Max: fmt.Sprintf("%d", to.UnixNano()),
		}).Result()
		if err != nil {
			return nil, errors.New("failed to get payment IDs: " + err.Error())
		}

		count := len(paymentIds)
		var totalAmount float64

		for _, paymentID := range paymentIds {
			paymentKey := "payment:" + paymentID
			amountStr, err := r.r.HGet(ctx, paymentKey, "amount").Result()
			if err != nil {
				continue
			}

			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				continue
			}

			totalAmount += amount
		}

		switch processor {
		case "default":
			summary.Default.Requests = count
			summary.Default.Amount = totalAmount
		case "fallback":
			summary.Fallback.Requests = count
			summary.Fallback.Amount = totalAmount
		}
	}

	return summary, nil
}

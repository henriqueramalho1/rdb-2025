package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const PaymentQueueName = "payments"

type PaymentsRepository struct {
	conn *pgxpool.Pool
	r    *redis.Client
}

func NewPaymentsRepository(conn *pgxpool.Pool, r *redis.Client) *PaymentsRepository {
	return &PaymentsRepository{
		conn: conn,
		r:    r,
	}
}

func (q *PaymentsRepository) Publish(ctx context.Context, data []byte) error {
	_, err := q.r.LPush(ctx, PaymentQueueName, data).Result()
	return err
}

func (q *PaymentsRepository) Consume(ctx context.Context) ([]byte, error) {
	result, err := q.r.BRPop(ctx, 0, PaymentQueueName).Result()
	if err != nil {
		return nil, err
	}

	data := []byte(result[1])
	return data, nil
}

func (r *PaymentsRepository) StorePayment(ctx context.Context, processor models.ProcessorType, payment *models.PaymentRequest) error {
	_, err := r.conn.Exec(ctx, "INSERT INTO payments (id, amount, processed_at, processor) VALUES ($1, $2, $3, $4)", payment.CorrelationId, payment.Amount, payment.RequestedAt, processor)
	if err != nil {
		return errors.New("failed to create payment: " + err.Error())
	}
	return nil
}

func (r *PaymentsRepository) GetPaymentsSummary(ctx context.Context, from, to time.Time) (*models.GlobalPaymentsSummary, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT processor, COUNT(*), SUM(amount)
		FROM payments
		WHERE processed_at BETWEEN $1 AND $2
		GROUP BY processor`, from, to)
	if err != nil {
		return nil, errors.New("failed to get payments summary: " + err.Error())
	}
	defer rows.Close()

	summary := &models.GlobalPaymentsSummary{}
	for rows.Next() {
		var processor string
		var count int
		var amount float64

		if err := rows.Scan(&processor, &count, &amount); err != nil {
			return nil, errors.New("failed to scan payment summary row: " + err.Error())
		}

		switch processor {
		case "default":
			summary.Default.Requests = count
			summary.Default.Amount = amount
		case "fallback":
			summary.Fallback.Requests = count
			summary.Fallback.Amount = amount
		default:
			return nil, errors.New("unknown processor type: " + processor)
		}
	}

	return summary, nil
}

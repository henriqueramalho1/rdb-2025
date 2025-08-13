package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentsRepository struct {
	conn *pgxpool.Pool
}

func NewPaymentsRepository(conn *pgxpool.Pool) *PaymentsRepository {
	return &PaymentsRepository{
		conn: conn,
	}
}

func (r *PaymentsRepository) Create(ctx context.Context, processor models.ProcessorType, payment *models.PaymentRequest) error {
	_, err := r.conn.Exec(ctx, "INSERT INTO payments (id, amount, processed_at, processor) VALUES ($1, $2, $3, $4)", payment.CorrelationId, payment.Amount, payment.RequestedAt, processor)
	if err != nil {
		return errors.New("failed to create payment: " + err.Error())
	}
	return nil
}

func (r *PaymentsRepository) GetPaymentsSummary(ctx context.Context, from, to time.Time, summary *models.GlobalPaymentsSummary) error {
	rows, err := r.conn.Query(ctx, `
		SELECT processor, COUNT(*), SUM(amount)
		FROM payments
		WHERE processed_at BETWEEN $1 AND $2
		GROUP BY processor`, from, to)
	if err != nil {
		return errors.New("failed to get payments summary: " + err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var processor string
		var count int
		var amount float64

		if err := rows.Scan(&processor, &count, &amount); err != nil {
			return errors.New("failed to scan payment summary row: " + err.Error())
		}

		switch processor {
		case "default":
			summary.Default.Requests = count
			summary.Default.Amount = amount
		case "fallback":
			summary.Fallback.Requests = count
			summary.Fallback.Amount = amount
		default:
			return errors.New("unknown processor type: " + processor)
		}
	}

	return nil
}

package services

import (
	"net/http"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
)

type PaymentService struct {
	httpClient    *http.Client
	paymentsQueue *queue.PaymentsQueue
}

func NewPaymentService(paymentsQueue *queue.PaymentsQueue) *PaymentService {
	return &PaymentService{
		httpClient:    &http.Client{},
		paymentsQueue: paymentsQueue,
	}
}

func (s *PaymentService) Process(payment *models.PaymentRequest) error {
	// decide if it is going to process in default or fallback based on status
	return nil
}

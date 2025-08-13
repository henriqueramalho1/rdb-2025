package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
)

type ProcessFailedError struct {
}

func (e *ProcessFailedError) Error() string {
	return "failed to process payment"
}

func NewProcessFailedError() *ProcessFailedError {
	return &ProcessFailedError{}
}

type PaymentsService struct {
	httpClient         *http.Client
	paymentsRepository *repositories.PaymentsRepository
}

func NewPaymentsService(paymentsRepository *repositories.PaymentsRepository) *PaymentsService {
	return &PaymentsService{
		httpClient:         &http.Client{},
		paymentsRepository: paymentsRepository,
	}
}

func (s *PaymentsService) Process(payment *models.PaymentRequest, processor models.ProcessorType) error {
	var url string
	switch processor {
	case models.DefaultProcessor:
		url = os.Getenv("DEFAULT_PROCESSOR_URL") + "/payments"
	case models.FallbackProcessor:
		url = os.Getenv("FALLBACK_PROCESSOR_URL") + "/payments"
	}

	payment.RequestedAt = time.Now().UTC()
	payload, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	log.Info("Processing payment in url ", url, " with payload: ", string(payload))
	response, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		defer response.Body.Close()
		log.Errorf("Failed to process payment in %s, status code %d and body %s", processor, response.StatusCode, string(body))
		return NewProcessFailedError()
	}

	err = s.paymentsRepository.Create(context.Background(), processor, payment)
	if err != nil {
		log.Error("Failed to create payment in repository")
		return err
	}
	log.Info("Payment processed successfully")
	return nil
}

func (s *PaymentsService) GetPaymentsSummary(ctx context.Context, from, to time.Time) (*models.GlobalPaymentsSummary, error) {
	var summary models.GlobalPaymentsSummary
	s.paymentsRepository.GetPaymentsSummary(ctx, from, to, &summary)

	return &summary, nil
}

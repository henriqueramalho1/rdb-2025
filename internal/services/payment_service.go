package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
)

type PaymentService struct {
	httpClient *http.Client
}

func NewPaymentService() *PaymentService {
	return &PaymentService{
		httpClient: &http.Client{},
	}
}

func (s *PaymentService) Process(payment *models.PaymentRequest, processor models.ProcessorType) error {
	var url string
	switch processor {
	case models.DefaultProcessor:
		url = os.Getenv("DEFAULT_PROCESSOR_URL") + "/payments"
	case models.FallbackProcessor:
		url = os.Getenv("FALLBACK_PROCESSOR_URL") + "/payments"
	}

	payload, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	response, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if response.StatusCode > 399 {
		return errors.New("processor returned status code " + response.Status)
	}

	return nil
}

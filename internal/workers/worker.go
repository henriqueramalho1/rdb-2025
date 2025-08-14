package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
)

var reqPool = sync.Pool{
	New: func() interface{} {
		return &models.PaymentRequest{}
	},
}

type PaymentWorker struct {
	config     *models.Config
	repo       *repositories.PaymentsRepository
	httpClient *http.Client
}

func NewPaymentWorker(config *models.Config, repo *repositories.PaymentsRepository) *PaymentWorker {
	return &PaymentWorker{
		config:     config,
		repo:       repo,
		httpClient: &http.Client{},
	}
}

func (w *PaymentWorker) Start(ctx context.Context) {
	for {
		data, err := w.repo.Consume(ctx)
		if err != nil {
			continue
		}

		req := reqPool.Get().(*models.PaymentRequest)
		if err := json.Unmarshal(data, req); err != nil {
			reqPool.Put(req)
			continue
		}

		err = w.process(ctx, models.DefaultProcessor, *req)
		if err == nil {
			continue
		}

		if err := w.process(ctx, models.FallbackProcessor, *req); err != nil {
			reqPool.Put(req)
			w.repo.Publish(ctx, data)
		}

		reqPool.Put(req)
	}
}

func (w *PaymentWorker) process(ctx context.Context, processor models.ProcessorType, req models.PaymentRequest) error {
	req.RequestedAt = time.Now().UTC()
	log.Infof("processing payment %s with requested at %s", req.CorrelationId, req.RequestedAt)
	data, err := json.Marshal(req)
	if err != nil {
		log.Errorf("failed to marshal payment request: %v", err)
		return err
	}

	var url string
	switch processor {
	case models.DefaultProcessor:
		url = w.config.DefaultUrl
	case models.FallbackProcessor:
		url = w.config.FallbackUrl
	default:
		return errors.New("unknown processor type")
	}

	resp, err := w.httpClient.Post(url+"/payments", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("failed to send payment request: %v", err)
		return err
	}

	if resp.StatusCode > 399 {
		return errors.New("failed to process payment, status code: " + resp.Status)
	}

	log.Infof("success processing payment %s with requested at %s", req.CorrelationId, req.RequestedAt)
	err = w.repo.StorePayment(ctx, processor, &req)

	if err != nil {
		log.Errorf("failed to store payment %s: %v", req.CorrelationId, err)
		return nil
	}

	return nil
}

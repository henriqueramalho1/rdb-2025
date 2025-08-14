package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand"
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
	config      *models.Config
	paymentRepo *repositories.PaymentsRepository
	healthRepo  *repositories.HealthRepository
	httpClient  *http.Client
}

func NewPaymentWorker(config *models.Config, paymentRepo *repositories.PaymentsRepository, healthRepo *repositories.HealthRepository) *PaymentWorker {
	return &PaymentWorker{
		config:      config,
		paymentRepo: paymentRepo,
		healthRepo:  healthRepo,
		httpClient:  &http.Client{},
	}
}

func (w *PaymentWorker) Start(ctx context.Context) {
	for {
		data, err := w.paymentRepo.Consume(ctx)
		if err != nil {
			continue
		}
		defaultFailing := w.healthRepo.IsProcessorFailing(ctx, models.DefaultProcessor)
		fallbackFailing := w.healthRepo.IsProcessorFailing(ctx, models.FallbackProcessor)

		req := reqPool.Get().(*models.PaymentRequest)
		if err := json.Unmarshal(data, req); err != nil {
			reqPool.Put(req)
			continue
		}

		bypass := rand.Intn(4) == 0

		if !defaultFailing || (bypass && defaultFailing && fallbackFailing) {
			err = w.process(ctx, models.DefaultProcessor, *req)
			if err == nil {
				reqPool.Put(req)
				continue
			}
		}

		if !fallbackFailing || (bypass && defaultFailing && fallbackFailing) {
			err = w.process(ctx, models.FallbackProcessor, *req)
			if err == nil {
				reqPool.Put(req)
				continue
			}
		}

		log.Info("fallback processor is failing, re-queuing payment")
		reqPool.Put(req)
		w.paymentRepo.Publish(ctx, data)
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
		log.Errorf("failed to process payment in %s, status code: %s", processor, resp.Status)
		go w.healthRepo.SetProcessorStatus(ctx, processor, false)
		return errors.New("failed to process payment, status code: " + resp.Status)
	}

	log.Infof("success processing payment %s with %s", req.CorrelationId, processor)
	go w.healthRepo.SetProcessorStatus(ctx, processor, true)
	err = w.paymentRepo.StorePayment(ctx, processor, &req)

	if err != nil {
		log.Errorf("failed to store payment %s: %v", req.CorrelationId, err)
		return nil
	}

	return nil
}

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

func NewPaymentWorker(config *models.Config, paymentRepo *repositories.PaymentsRepository, healthRepo *repositories.HealthRepository, httpClient *http.Client) *PaymentWorker {
	return &PaymentWorker{
		config:      config,
		paymentRepo: paymentRepo,
		healthRepo:  healthRepo,
		httpClient:  httpClient,
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

		preferredProcessor := models.DefaultProcessor

		if !defaultFailing && !fallbackFailing {
			preferredProcessor = w.getPreferredProcessor(ctx)
			log.Infof("custom preferred processor is %s", preferredProcessor)
		}

		req := reqPool.Get().(*models.PaymentRequest)
		if err := json.Unmarshal(data, req); err != nil {
			reqPool.Put(req)
			continue
		}

		bypass := rand.Intn(4) == 0

		/*
			process in default:
			- only option available
			- if both are available and the preferred processor is default
			- if both are failing and rand bypass allows it to try to discover if system is on
		*/
		if (!defaultFailing && fallbackFailing) || (!defaultFailing && !fallbackFailing && preferredProcessor == models.DefaultProcessor) || (defaultFailing && bypass) {
			err = w.process(ctx, models.DefaultProcessor, *req)
			if err == nil {
				reqPool.Put(req)
				continue
			}
			log.Info("default processor is failing")
		}

		if fallbackFailing && !defaultFailing {
			bypass = rand.Intn(8) == 0 // in case fallback is failing, decreases the chances of bypass
		}

		/*
			process in default:
			- only option available
			- if both are available and the preferred processor is fallback
			- if both are failing and rand bypass allows it to try to discover if system is on
		*/
		if (!fallbackFailing && !defaultFailing) || (!defaultFailing && !fallbackFailing && preferredProcessor == models.FallbackProcessor) || (fallbackFailing && bypass) {
			err = w.process(ctx, models.FallbackProcessor, *req)
			if err == nil {
				reqPool.Put(req)
				continue
			}
			log.Info("fallback processor is failing")
		}

		log.Infof("re-queuing payment %s", req.CorrelationId)
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

	defer resp.Body.Close()

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

func (w *PaymentWorker) getPreferredProcessor(ctx context.Context) models.ProcessorType {
	defaultMinResponseTime := w.healthRepo.GetProcessorMinResponseTime(ctx, models.DefaultProcessor)
	fallbackMinResponseTime := w.healthRepo.GetProcessorMinResponseTime(ctx, models.FallbackProcessor)

	defaultTax := 0.05  // 5%
	fallbackTax := 0.15 // 15%

	defaultCost := float64(defaultMinResponseTime) * (1 + defaultTax)
	fallbackCost := float64(fallbackMinResponseTime) * (1 + fallbackTax)

	thresholdPercent := 100.0
	if fallbackCost < defaultCost*(thresholdPercent/100.0) {
		return models.FallbackProcessor
	}

	return models.DefaultProcessor
}

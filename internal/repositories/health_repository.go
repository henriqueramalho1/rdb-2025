package repositories

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
)

type HealthRepository struct {
	httpClient *http.Client
	client     *redis.Client
	config     *models.Config
}

func NewHealthRepository(httpClient *http.Client, client *redis.Client, config *models.Config) *HealthRepository {
	return &HealthRepository{
		httpClient: httpClient,
		client:     client,
		config:     config,
	}
}

func (r *HealthRepository) HealthCheckTask(ctx context.Context) {

	result, _ := r.client.SetNX(ctx, "health_check", "ok", 0).Result()
	if !result {
		log.Info("another instance is the owner of the health check task")
		return
	}

	ch := make(chan models.ProcessorStatus, 5)

	defaultTicker := time.NewTicker(5 * time.Second)

	go func() {
		for range defaultTicker.C {
			go r.getStatusAndMinResponseTime(models.DefaultProcessor, ch)
		}
	}()

	time.Sleep(2500 * time.Millisecond)
	fallbackTicker := time.NewTicker(5 * time.Second)

	go func() {
		for range fallbackTicker.C {
			go r.getStatusAndMinResponseTime(models.FallbackProcessor, ch)
		}
	}()

	for status := range ch {
		log.Infof("received health status from %s processor as %v", status.Processor, status)
		r.SetProcessorStatus(ctx, status.Processor, !status.Failing)
		r.SetProcessorMinResponseTime(ctx, status.Processor, status.MinResponseTime)
	}
}

func (r *HealthRepository) getStatusAndMinResponseTime(processor models.ProcessorType, ch chan<- models.ProcessorStatus) {
	var url string
	switch processor {
	case models.DefaultProcessor:
		url = r.config.DefaultUrl
	case models.FallbackProcessor:
		url = r.config.FallbackUrl
	}

	resp, err := r.httpClient.Get(url + "/payments/service-health")
	if err != nil {
		log.Errorf("failed get processor %s status: %v", processor, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Warnf("health check detected processor %s is overloaded", processor)
		return
	}

	if resp.StatusCode > 399 {
		log.Info("health check detected processor %s is failing, status code: %s", processor, resp.Status)
		ch <- models.ProcessorStatus{
			Processor: processor,
			Failing:   true,
		}
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response body of health check route: %v", err)
		return
	}

	var status models.ProcessorStatus
	json.Unmarshal(data, &status)
	status.Processor = processor

	ch <- status
}

func (r *HealthRepository) IsProcessorFailing(ctx context.Context, processor models.ProcessorType) bool {
	_, err := r.client.Get(ctx, string(processor)+":failing").Result()
	if err == redis.Nil {
		return false // If the key does not exist, we assume the processor is not failing
	}

	if err != nil {
		return false
	}

	return true
}

func (r *HealthRepository) SetProcessorStatus(ctx context.Context, processor models.ProcessorType, on bool) {
	if on {
		r.client.Del(ctx, string(processor)+":failing").Err()
		return
	}

	isFailingStr := "true"
	r.client.Set(ctx, string(processor)+":failing", isFailingStr, 0).Err()
}

func (r *HealthRepository) GetProcessorMinResponseTime(ctx context.Context, processor models.ProcessorType) int {
	val, err := r.client.Get(ctx, string(processor)+":min_response_time").Result()
	if err == redis.Nil {
		return 0
	}
	if err != nil {
		return 0
	}

	minResponseTime, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return minResponseTime
}

func (r *HealthRepository) SetProcessorMinResponseTime(ctx context.Context, processor models.ProcessorType, minResponseTime int) {
	r.client.Set(ctx, string(processor)+":min_response_time", minResponseTime, 0).Err()
}

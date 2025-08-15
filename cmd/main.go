package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/handlers"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
	"github.com/henriqueramalho1/rdb-2025/internal/workers"
	"github.com/redis/go-redis/v9"
)

func main() {
	time.Sleep(5 * time.Second)

	ctx := context.Background()
	s := fiber.New(fiber.Config{Immutable: true})
	r := getRedisConnection()
	defer r.Close()

	c := getConfig()
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	paymentsRepo := repositories.NewPaymentsRepository(r)
	healthRepo := repositories.NewHealthRepository(httpClient, r, c)

	paymentsHandler := handlers.NewPaymentsHandler(paymentsRepo)
	worker := workers.NewPaymentWorker(c, paymentsRepo, healthRepo, httpClient)

	s.Get("/health", handlers.HealthCheck)
	s.Get("/payments-summary", paymentsHandler.PaymentsSummary)
	s.Post("/payments", paymentsHandler.CreatePayment)

	go healthRepo.HealthCheckTask(ctx)

	for i := 0; i < c.NumWorkers; i++ {
		go worker.Start(ctx)
	}

	s.Listen(":8080")
}

func getConfig() *models.Config {
	workers := os.Getenv("NUM_WORKERS")
	numWorkers, err := strconv.Atoi(workers)

	if err != nil || numWorkers <= 0 {
		log.Fatal("invalid NUM_WORKERS value:", workers)
	}

	return &models.Config{
		DefaultUrl:  os.Getenv("DEFAULT_PROCESSOR_URL"),
		FallbackUrl: os.Getenv("FALLBACK_PROCESSOR_URL"),
		NumWorkers:  numWorkers,
	}
}

func getRedisConnection() *redis.Client {
	ctx := context.Background()
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := fmt.Sprintf("%s:%s", host, port)

	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		PoolSize:        20, // Increase pool size for high load
		MinIdleConns:    5,  // Keep connections ready
		MaxIdleConns:    10,
		ConnMaxIdleTime: time.Minute,
		DialTimeout:     50 * time.Millisecond,
		ReadTimeout:     50 * time.Millisecond,
		WriteTimeout:    50 * time.Millisecond,
		PoolTimeout:     100 * time.Millisecond,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to redis:", err)
		return nil
	}
	return client
}

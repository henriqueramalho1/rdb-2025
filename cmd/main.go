package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/henriqueramalho1/rdb-2025/internal/handlers"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
	"github.com/henriqueramalho1/rdb-2025/internal/services"
	"github.com/henriqueramalho1/rdb-2025/internal/workers"
	"github.com/redis/go-redis/v9"
)

func main() {
	s := fiber.New(fiber.Config{Immutable: true})
	r := getRedisConnection()
	defer r.Close()

	paymentsQueue := queue.NewPaymentsQueue(r)
	healthRepo := repositories.NewHealthRepository(r)

	healthService := services.NewHealthCheckerService(healthRepo)
	paymentService := services.NewPaymentService()
	paymentsHandler := handlers.NewPaymentsHandler(paymentsQueue)
	worker := workers.NewPaymentWorker(healthService, paymentService, paymentsQueue)

	s.Get("/health", handlers.HealthCheck)
	s.Post("/payments", paymentsHandler.CreatePayment)

	for i := 0; i < 16; i++ {
		go worker.ProcessPayment()
	}

	s.Listen(":8080")
}

func getRedisConnection() *redis.Client {
	ctx := context.Background()
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := fmt.Sprintf("%s:%s", host, port)

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to redis:", err)
		return nil
	}
	return client
}

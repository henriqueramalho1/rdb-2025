package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/henriqueramalho1/rdb-2025/internal/handlers"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/redis/go-redis/v9"
)

func main() {
	s := fiber.New(fiber.Config{Immutable: true})
	r := getRedisConnection()
	defer r.Close()

	paymentsQueue := queue.NewPaymentsQueue(r)
	paymentsHandler := handlers.NewPaymentsHandler(paymentsQueue)

	s.Get("/health", handlers.HealthCheck)
	s.Post("/payments", paymentsHandler.CreatePayment)

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

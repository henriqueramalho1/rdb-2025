package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/handlers"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
	"github.com/henriqueramalho1/rdb-2025/internal/services"
	"github.com/henriqueramalho1/rdb-2025/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	time.Sleep(5 * time.Second)
	s := fiber.New(fiber.Config{Immutable: true})
	p := getPostgresConnection()
	r := getRedisConnection()
	defer r.Close()

	paymentsQueue := queue.NewPaymentsQueue(r)
	healthRepo := repositories.NewHealthRepository(r)
	paymentsRepo := repositories.NewPaymentsRepository(p)

	healthService := services.NewHealthCheckerService(healthRepo)
	paymentService := services.NewPaymentsService(paymentsRepo)
	paymentsHandler := handlers.NewPaymentsHandler(paymentService, paymentsQueue)
	worker := workers.NewPaymentWorker(healthService, paymentService, paymentsQueue)

	s.Get("/health", handlers.HealthCheck)
	s.Get("/payments-summary", paymentsHandler.PaymentsSummary)
	s.Post("/payments", paymentsHandler.CreatePayment)

	for i := 0; i < 16; i++ {
		go worker.ProcessPayment()
	}

	s.Listen(":8080")
}

func getPostgresConnection() *pgxpool.Pool {
	ctx := context.Background()
	host := os.Getenv("POSTGRES_HOST")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s", user, password, host, dbname)
	log.Info("Connecting to postgres:", dsn)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping postgres:", err)
		return nil
	}

	return pool
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

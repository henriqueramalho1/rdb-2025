package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/handlers"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
	"github.com/henriqueramalho1/rdb-2025/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	time.Sleep(5 * time.Second)

	ctx := context.Background()
	s := fiber.New(fiber.Config{Immutable: true})
	p := getPostgresConnection()
	r := getRedisConnection()
	defer r.Close()

	c := getConfig()
	paymentsRepo := repositories.NewPaymentsRepository(p, r)
	healthRepo := repositories.NewHealthRepository(r)

	paymentsHandler := handlers.NewPaymentsHandler(paymentsRepo)
	worker := workers.NewPaymentWorker(c, paymentsRepo, healthRepo)

	s.Get("/health", handlers.HealthCheck)
	s.Get("/payments-summary", paymentsHandler.PaymentsSummary)
	s.Post("/payments", paymentsHandler.CreatePayment)

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

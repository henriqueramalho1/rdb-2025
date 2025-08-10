package queue

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testRedisClient is a package-level client initialized in TestMain and
// can be used by tests in this package.
var testRedisClient *redis.Client

// testContainer holds the running Redis container reference for cleanup.
var testContainer tc.Container

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start a temporary Redis container.
	req := tc.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start redis container: %v\n", err)
		os.Exit(1)
	}
	testContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	_ = os.Setenv("TEST_REDIS_ADDR", addr)
	testRedisClient = redis.NewClient(&redis.Options{Addr: addr})

	if err := testRedisClient.Ping(ctx).Err(); err != nil {
		_ = container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to ping redis: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testRedisClient != nil {
		_ = testRedisClient.Close()
	}
	_ = container.Terminate(ctx)

	os.Exit(code)
}

func TestPaymentsQueue(t *testing.T) {
	ctx := context.Background()
	queue := NewPaymentsQueue(testRedisClient)

	t.Run("Publish and Consume", func(t *testing.T) {
		msg := &models.PaymentRequest{CorrelationId: "1", Amount: float64(23.2)}
		err := queue.Publish(ctx, msg)
		if err != nil {
			t.Fatalf("failed to publish message: %v", err)
		}

		request, err := queue.Consume(ctx)
		if err != nil {
			t.Fatalf("failed to consume message: %v", err)
		}

		if request.CorrelationId != msg.CorrelationId {
			t.Errorf("unexpected message: got %+v, want %+v", request.CorrelationId, msg.CorrelationId)
		}
	})
}

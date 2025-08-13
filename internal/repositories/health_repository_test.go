package repositories

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
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

func TestHealthRepository(t *testing.T) {
	repo := NewHealthRepository(testRedisClient)

	t.Run("Set Coordinator Flag", func(t *testing.T) {
		success, err := repo.SetCoordinatorFlag()
		if err != nil {
			t.Fatalf("failed to set coordinator flag: %v", err)
		}
		if !success {
			t.Error("expected coordinator flag to be set")
		}
	})

	t.Run("Set And Get Processor Status", func(t *testing.T) {
		status := models.ProcessorStatus{Failing: false, MinResponseTime: 100}
		err := repo.SetProcessorStatus(models.DefaultProcessor, status)
		if err != nil {
			t.Fatalf("failed to set processor status: %v", err)
		}
		status, err = repo.GetProcessorStatus(models.DefaultProcessor)
		if err != nil {
			t.Fatalf("failed to get processor status: %v", err)
		}
		if status.Failing {
			t.Error("expected processor to be healthy")
		}

		assert.Equal(t, status, models.ProcessorStatus{Failing: false, MinResponseTime: 100})
	})
}

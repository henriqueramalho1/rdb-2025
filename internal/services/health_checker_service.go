package services

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
)

type HealthCheckerService struct {
	httpClient *http.Client
}

func NewHealthCheckerService() *HealthCheckerService {
	return &HealthCheckerService{
		httpClient: &http.Client{},
	}
}

func (s *HealthCheckerService) CheckDefaultProcessorHealth() {

	for {
		resp, err := s.httpClient.Get(os.Getenv("DEFAULT_PROCESSOR_URL") + "/health")
		if err != nil {
			log.Error("Error checking default processor health:", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Error("Default processor health check failed with status:", resp.StatusCode)
			time.Sleep(8 * time.Second)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Error("Error reading default processor health response:", err)
			continue
		}

		var processorStatus models.HealthStatus
		err = json.Unmarshal(respBody, &processorStatus)
		if err != nil {
			log.Error("Error unmarshalling default processor health response:", err)
			continue
		}

		time.Sleep(5 * time.Second)
	}
}

func (s *HealthCheckerService) FetchProcessorsStatus() error {
	// make the request to default and fallback and update cache
	return nil
}

func (s *HealthCheckerService) GetProcessorsStatus() models.HealthStatus {
	// retrieve status from cache
	return models.HealthStatus{}
}

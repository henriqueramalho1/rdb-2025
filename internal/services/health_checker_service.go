package services

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
)

type HealthCheckerService struct {
	httpClient       *http.Client
	healthRepository *repositories.HealthRepository
}

func NewHealthCheckerService(healthRepo *repositories.HealthRepository) *HealthCheckerService {
	return &HealthCheckerService{
		httpClient:       &http.Client{},
		healthRepository: healthRepo,
	}
}

func (s *HealthCheckerService) makeRequest(url string) (models.ProcessorStatus, error) {
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return models.ProcessorStatus{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return models.ProcessorStatus{}, errors.New("failed to get valid response from processor, status code " + resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return models.ProcessorStatus{}, err
	}

	var status models.ProcessorStatus
	err = json.Unmarshal(respBody, &status)
	if err != nil {
		return models.ProcessorStatus{}, err
	}

	return status, nil
}

func (s *HealthCheckerService) GetProcessorsStatus() models.HealthStatus {
	defaultStatus, err := s.healthRepository.GetProcessorStatus(models.DefaultProcessor)
	if err != nil {
		log.Error("Error fetching default processor status:", err)
		return models.HealthStatus{}
	}

	fallbackStatus, err := s.healthRepository.GetProcessorStatus(models.FallbackProcessor)
	if err != nil {
		log.Error("Error fetching fallback processor status:", err)
		return models.HealthStatus{}
	}

	return models.HealthStatus{
		Default:  defaultStatus,
		Fallback: fallbackStatus,
	}
}

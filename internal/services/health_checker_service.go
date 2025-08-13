package services

import (
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

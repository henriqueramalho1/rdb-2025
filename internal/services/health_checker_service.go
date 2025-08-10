package services

import (
	"net/http"

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

func (s *HealthCheckerService) FetchProcessorsStatus() error {
	// make the request to default and fallback and update cache
	return nil
}

func (s *HealthCheckerService) GetProcessorsStatus() *models.ProcessorsStatus {
	// retrieve status from cache
	return nil
}

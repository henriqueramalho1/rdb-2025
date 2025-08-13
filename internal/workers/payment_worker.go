package workers

import (
	"context"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/henriqueramalho1/rdb-2025/internal/services"
)

type PaymentWorker struct {
	healthService   *services.HealthCheckerService
	paymentsService *services.PaymentsService
	paymentsQueue   *queue.PaymentsQueue
}

func NewPaymentWorker(healthService *services.HealthCheckerService, paymentsService *services.PaymentsService, paymentsQueue *queue.PaymentsQueue) *PaymentWorker {
	return &PaymentWorker{
		healthService:   healthService,
		paymentsService: paymentsService,
		paymentsQueue:   paymentsQueue,
	}
}

func (w *PaymentWorker) ProcessPayment() {
	for {
		request, err := w.paymentsQueue.Consume(context.Background())
		if err != nil {
			continue
		}

		status := w.healthService.GetProcessorsStatus()
		if !status.Default.Failing {
			log.Info("Processing in default processor")
			err := w.paymentsService.Process(request, models.DefaultProcessor)
			if err == nil {
				continue
			}
		}

		if !status.Fallback.Failing {
			log.Info("Processing in fallback processor")
			err := w.paymentsService.Process(request, models.FallbackProcessor)
			if err == nil {
				continue
			}
		}

		log.Info("Couldnt process payment, re-enqueuing")
		w.paymentsQueue.Publish(context.Background(), request)
	}
}

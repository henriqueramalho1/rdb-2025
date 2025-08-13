package workers

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2/log"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/henriqueramalho1/rdb-2025/internal/services"
)

type PaymentWorker struct {
	healthService  *services.HealthCheckerService
	paymentService *services.PaymentService
	paymentsQueue  *queue.PaymentsQueue
}

func NewPaymentWorker(healthService *services.HealthCheckerService, paymentService *services.PaymentService, paymentsQueue *queue.PaymentsQueue) *PaymentWorker {
	return &PaymentWorker{
		healthService:  healthService,
		paymentService: paymentService,
		paymentsQueue:  paymentsQueue,
	}
}

func (w *PaymentWorker) ProcessPayment() {
	for {
		request, err := w.paymentsQueue.Consume(context.Background())
		if err != nil {
			continue
		}

		processorsStatus := w.healthService.GetProcessorsStatus()

		if processorsStatus.Default.Failing {
			err = w.paymentService.Process(request, models.FallbackProcessor)
			if err != nil {
				log.Error("Failed to process payment with fallback processor: ", err)
			}
		}

		err = w.paymentService.Process(request, models.DefaultProcessor)
		if err != nil {
			log.Error("Failed to process payment with default processor: ", err)
			var processFailedErr *services.ProcessFailedError
			if errors.As(err, &processFailedErr) {
				w.healthService.SetProcessorStatus(models.DefaultProcessor, models.ProcessorStatus{Failing: true})
				log.Info("Marked default processor as failing")
			}
		}
	}
}

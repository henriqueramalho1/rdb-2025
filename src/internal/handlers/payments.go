package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
	"github.com/henriqueramalho1/rdb-2025/internal/services"
)

type PaymentsHandler struct {
	paymentsService *services.PaymentsService
	paymentsQueue   *queue.PaymentsQueue
}

func NewPaymentsHandler(paymentService *services.PaymentsService, paymentsQueue *queue.PaymentsQueue) *PaymentsHandler {
	return &PaymentsHandler{
		paymentsService: paymentService,
		paymentsQueue:   paymentsQueue,
	}
}

func (h *PaymentsHandler) CreatePayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	go h.paymentsQueue.Publish(c.Context(), &req)
	return c.SendStatus(http.StatusOK)
}

func (h *PaymentsHandler) PaymentsSummary(c *fiber.Ctx) error {
	from := c.Query("from")
	to := c.Query("to")

	timeLayout := "2006-01-02T15:04:05Z07:00"
	fromTime, err := time.Parse(timeLayout, from)
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}
	toTime, err := time.Parse(timeLayout, to)
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	summary, err := h.paymentsService.GetPaymentsSummary(c.Context(), fromTime, toTime)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.JSON(summary)
}

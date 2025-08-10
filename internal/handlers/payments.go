package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/henriqueramalho1/rdb-2025/internal/models"
	"github.com/henriqueramalho1/rdb-2025/internal/queue"
)

type PaymentsHandler struct {
	paymentsQueue *queue.PaymentsQueue
}

func NewPaymentsHandler(paymentsQueue *queue.PaymentsQueue) *PaymentsHandler {
	return &PaymentsHandler{paymentsQueue: paymentsQueue}
}

func (h *PaymentsHandler) CreatePayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	h.paymentsQueue.Publish(c.Context(), &req)
	return c.SendStatus(http.StatusOK)
}

func (h *PaymentsHandler) PaymentsSummary(c *fiber.Ctx) error {
	return nil
}

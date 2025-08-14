package handlers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/henriqueramalho1/rdb-2025/internal/repositories"
)

type PaymentsHandler struct {
	repo *repositories.PaymentsRepository
}

func NewPaymentsHandler(repo *repositories.PaymentsRepository) *PaymentsHandler {
	return &PaymentsHandler{
		repo: repo,
	}
}

func (h *PaymentsHandler) CreatePayment(c *fiber.Ctx) error {
	data := c.Body()
	h.repo.Publish(c.Context(), data)
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

	summary, err := h.repo.GetPaymentsSummary(c.Context(), fromTime, toTime)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.JSON(summary)
}

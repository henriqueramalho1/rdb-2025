package models

import "time"

type ProcessorType string

const (
	DefaultProcessor  ProcessorType = "default"
	FallbackProcessor ProcessorType = "fallback"
)

type PaymentRequest struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}

type ProcessorStatus struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type PaymentsSummary struct {
	Requests int     `json:"totalRequests"`
	Amount   float64 `json:"totalAmount"`
}

type GlobalPaymentsSummary struct {
	Default  PaymentsSummary `json:"default"`
	Fallback PaymentsSummary `json:"fallback"`
}

type HealthStatus struct {
	Default  ProcessorStatus
	Fallback ProcessorStatus
}

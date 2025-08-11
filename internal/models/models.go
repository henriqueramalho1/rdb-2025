package models

type ProcessorType string

const (
	DefaultProcessor  ProcessorType = "default"
	FallbackProcessor ProcessorType = "fallback"
)

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type ProcessorStatus struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

type HealthStatus struct {
	Default  ProcessorStatus
	Fallback ProcessorStatus
}

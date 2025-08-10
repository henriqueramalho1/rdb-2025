package models

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}

type ProcessorsStatus struct {
	IsFallback      bool `json:"isFallback"`
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

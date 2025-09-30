package whatsmiau

import "time"

// StartPairingResponse representa a resposta do início do pairing
type StartPairingResponse struct {
	Success     bool   `json:"success"`
	PairingCode string `json:"pairingCode,omitempty"` // XXXX-XXXX
	Message     string `json:"message"`
	SessionID   string `json:"sessionId,omitempty"`
	ExpiresAt   int64  `json:"expiresAt,omitempty"`
}

// PairingStatusResponse representa a resposta do status do pairing
type PairingStatusResponse struct {
	Status    string `json:"status"` // "pending", "success", "failed", "expired"
	Message   string `json:"message"`
	SessionID string `json:"sessionId,omitempty"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
}

// PairingSession representa uma sessão de pairing
type PairingSession struct {
	ID          string    `json:"id"`
	InstanceID  string    `json:"instanceId"`
	PhoneNumber string    `json:"phoneNumber"`
	Code        string    `json:"code"`
	ClientType  string    `json:"clientType"`
	ClientName  string    `json:"clientName"`
	Status      string    `json:"status"` // "pending", "success", "failed", "expired"
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

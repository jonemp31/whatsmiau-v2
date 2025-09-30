package dto

type SendStatusTextRequest struct {
	InstanceID string `json:"instanceId" param:"instance" validate:"required"`
	Text       string `json:"text" validate:"required"`
	Background string `json:"background,omitempty"` // Cor de fundo (ex: #FF0000)
	Font       string `json:"font,omitempty"`       // Fonte do texto
}

type SendStatusMediaRequest struct {
	InstanceID string `json:"instanceId" param:"instance" validate:"required"`
	Media      string `json:"media" validate:"required"` // URL da mídia
	Caption    string `json:"caption,omitempty"`         // Legenda
	Mimetype   string `json:"mimetype,omitempty"`        // Tipo MIME
}

type SendStatusResponse struct {
	ID          string `json:"id"`
	CreatedAt   int64  `json:"createdAt"`
	InstanceID  string `json:"instanceId"`
	MessageType string `json:"messageType"`
	Status      string `json:"status"`
}

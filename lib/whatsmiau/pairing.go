package whatsmiau

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.uber.org/zap"
)

// StartPairing inicia o processo de pairing para uma instância
func (s *Whatsmiau) StartPairing(ctx context.Context, instanceID, phoneNumber, clientType, clientName string) (*StartPairingResponse, error) {
	// Verificar se a instância existe
	instance := s.getInstanceCached(instanceID)
	if instance == nil {
		return nil, fmt.Errorf("instance not found")
	}

	// Verificar se já está conectada
	client, ok := s.clients.Load(instanceID)
	if ok && client.IsLoggedIn() {
		return &StartPairingResponse{
			Success: false,
			Message: "Instance is already connected",
		}, nil
	}

	// Mapear tipo de cliente
	pairClientType := s.mapClientType(clientType)
	if clientName == "" {
		clientName = s.mapClientName(clientType)
	}

	// Gerar ID de sessão único
	sessionID := s.generateSessionID()

	// Gerar código de pairing
	code, err := s.generatePairingCode(ctx, instanceID, phoneNumber, pairClientType, clientName)
	if err != nil {
		zap.L().Error("Failed to generate pairing code",
			zap.String("instance", instanceID),
			zap.String("phone", phoneNumber),
			zap.Error(err),
		)
		return nil, err
	}

	// Criar sessão de pairing
	expiresAt := time.Now().Add(160 * time.Second) // 160 segundos de expiração
	session := PairingSession{
		ID:          sessionID,
		InstanceID:  instanceID,
		PhoneNumber: phoneNumber,
		Code:        code,
		ClientType:  clientType,
		ClientName:  clientName,
		Status:      "pending",
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	// Salvar no cache
	s.pairingCache.Store(sessionID, session)

	// Iniciar observação do pairing
	go s.observePairing(client, instanceID, sessionID)

	// Configurar limpeza automática
	go s.cleanupPairingSession(sessionID, 160*time.Second)

	zap.L().Info("Pairing Started",
		zap.String("instance", instanceID),
		zap.String("phone", phoneNumber),
		zap.String("session_id", sessionID),
		zap.String("code", code),
	)

	return &StartPairingResponse{
		Success:     true,
		PairingCode: code,
		Message:     "Pairing code generated. Check your WhatsApp for notification.",
		SessionID:   sessionID,
		ExpiresAt:   expiresAt.Unix(),
	}, nil
}

// GetPairingStatus retorna o status atual do pairing
func (s *Whatsmiau) GetPairingStatus(ctx context.Context, instanceID, sessionID string) (*PairingStatusResponse, error) {
	// Buscar sessão no cache
	session, ok := s.pairingCache.Load(sessionID)
	if !ok {
		return &PairingStatusResponse{
			Status:  "expired",
			Message: "Session not found or expired",
		}, nil
	}

	// Verificar se a instância está conectada
	client, ok := s.clients.Load(instanceID)
	if ok && client.IsLoggedIn() {
		// Atualizar status para sucesso
		session.Status = "success"
		s.pairingCache.Store(sessionID, session)

		zap.L().Info("Pairing Successful",
			zap.String("instance", instanceID),
			zap.String("session_id", sessionID),
			zap.String("phone", session.PhoneNumber),
		)

		return &PairingStatusResponse{
			Status:    "success",
			Message:   "Pairing successful, client is logged in",
			SessionID: sessionID,
			ExpiresAt: session.ExpiresAt.Unix(),
		}, nil
	}

	// Verificar se expirou
	if time.Now().After(session.ExpiresAt) {
		session.Status = "expired"
		s.pairingCache.Store(sessionID, session)

		zap.L().Info("Pairing Expired",
			zap.String("instance", instanceID),
			zap.String("session_id", sessionID),
		)

		return &PairingStatusResponse{
			Status:    "expired",
			Message:   "Pairing code has expired",
			SessionID: sessionID,
			ExpiresAt: session.ExpiresAt.Unix(),
		}, nil
	}

	// Retornar status atual
	message := s.getPairingStatusMessage(session.Status)
	return &PairingStatusResponse{
		Status:    session.Status,
		Message:   message,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt.Unix(),
	}, nil
}

// generatePairingCode gera o código de pairing usando o Whatsmeow
func (s *Whatsmiau) generatePairingCode(ctx context.Context, instanceID, phoneNumber string, clientType whatsmeow.PairClientType, clientName string) (string, error) {
	// Obter ou criar cliente
	client, ok := s.clients.Load(instanceID)
	if !ok {
		device := s.container.NewDevice()
		client = whatsmeow.NewClient(device, s.logger)
		s.clients.Store(instanceID, client)
	}

	// Conectar se necessário
	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			return "", err
		}
	}

	// Gerar código de pairing
	code, err := client.PairPhone(
		ctx,
		phoneNumber,
		true, // showPushNotification
		clientType,
		clientName,
	)
	if err != nil {
		return "", err
	}

	zap.L().Info("Pairing Code Generated",
		zap.String("instance", instanceID),
		zap.String("phone", phoneNumber),
		zap.String("code", code),
	)

	return code, nil
}

// observePairing observa o status do pairing
func (s *Whatsmiau) observePairing(client *whatsmeow.Client, instanceID, sessionID string) {
	if _, ok := s.pairingObserver.Load(instanceID); ok {
		return
	}
	s.pairingObserver.Store(instanceID, true)
	defer func() {
		s.pairingObserver.Delete(instanceID)
	}()

	// Aguardar conexão ou timeout
	timeout := time.After(160 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout - marcar como expirado
			if session, ok := s.pairingCache.Load(sessionID); ok {
				session.Status = "expired"
				s.pairingCache.Store(sessionID, session)
			}
			return

		case <-ticker.C:
			// Verificar se está conectado
			if client.IsLoggedIn() {
				// Sucesso - atualizar status
				if session, ok := s.pairingCache.Load(sessionID); ok {
					session.Status = "success"
					s.pairingCache.Store(sessionID, session)

					zap.L().Info("Pairing Observer - Success",
						zap.String("instance", instanceID),
						zap.String("session_id", sessionID),
					)
				}
				return
			}
		}
	}
}

// cleanupPairingSession remove sessões expiradas
func (s *Whatsmiau) cleanupPairingSession(sessionID string, duration time.Duration) {
	time.Sleep(duration)
	if session, exists := s.pairingCache.Load(sessionID); exists {
		if session.Status == "pending" {
			session.Status = "expired"
			s.pairingCache.Store(sessionID, session)
		}
		// Remover do cache após um tempo adicional
		time.Sleep(30 * time.Second)
		s.pairingCache.Delete(sessionID)
	}
}

// mapClientType mapeia string para PairClientType
func (s *Whatsmiau) mapClientType(clientType string) whatsmeow.PairClientType {
	switch strings.ToLower(clientType) {
	case "chrome":
		return whatsmeow.PairClientChrome
	case "firefox":
		return whatsmeow.PairClientFirefox
	case "safari":
		return whatsmeow.PairClientSafari
	case "edge":
		return whatsmeow.PairClientEdge
	default:
		return whatsmeow.PairClientChrome
	}
}

// mapClientName mapeia tipo de cliente para nome padrão
func (s *Whatsmiau) mapClientName(clientType string) string {
	switch strings.ToLower(clientType) {
	case "chrome":
		return "Chrome (Windows)"
	case "firefox":
		return "Firefox (Windows)"
	case "safari":
		return "Safari (macOS)"
	case "edge":
		return "Edge (Windows)"
	default:
		return "Chrome (Windows)"
	}
}

// generateSessionID gera ID único para sessão
func (s *Whatsmiau) generateSessionID() string {
	return fmt.Sprintf("pairing_%s", time.Now().Format("20060102150405"))
}

// getPairingStatusMessage retorna mensagem descritiva do status
func (s *Whatsmiau) getPairingStatusMessage(status string) string {
	switch status {
	case "pending":
		return "Waiting for user to enter pairing code in WhatsApp"
	case "success":
		return "Pairing successful, client is logged in"
	case "failed":
		return "Pairing failed"
	case "expired":
		return "Pairing code has expired"
	default:
		return "Unknown status"
	}
}

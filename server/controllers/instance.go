package controllers

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/verbeux-ai/whatsmiau/lib/whatsmiau"
	"github.com/verbeux-ai/whatsmiau/models"
	"github.com/verbeux-ai/whatsmiau/repositories/instances"
	"go.mau.fi/whatsmeow/types"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/verbeux-ai/whatsmiau/interfaces"
	"github.com/verbeux-ai/whatsmiau/server/dto"
	"github.com/verbeux-ai/whatsmiau/utils"
	"go.uber.org/zap"
)

type Instance struct {
	repo      interfaces.InstanceRepository
	whatsmiau *whatsmiau.Whatsmiau
}

func NewInstances(repository interfaces.InstanceRepository, whatsmiau *whatsmiau.Whatsmiau) *Instance {
	return &Instance{
		repo:      repository,
		whatsmiau: whatsmiau,
	}
}

func (s *Instance) Create(ctx echo.Context) error {
	var request dto.CreateInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	request.ID = request.InstanceName
	if request.Instance == nil {
		request.Instance = &models.Instance{
			ID:               request.InstanceName,
			AutoReadMessages: true, // Ativar confirmações por padrão
			ReadDelay:        8,    // Delay padrão de 8 segundos
		}
	} else {
		request.Instance.ID = request.InstanceName
		// Se não especificado, ativar confirmações por padrão
		if !request.Instance.AutoReadMessages && request.Instance.ReadDelay == 0 {
			request.Instance.AutoReadMessages = true
			request.Instance.ReadDelay = 8
		}
	}
	request.RemoteJID = ""

	c := ctx.Request().Context()
	if err := s.repo.Create(c, request.Instance); err != nil {
		zap.L().Error("failed to create instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to create instance")
	}

	return ctx.JSON(http.StatusCreated, dto.CreateInstanceResponse{
		Instance: request.Instance,
	})
}

func (s *Instance) Update(ctx echo.Context) error {
	var request dto.UpdateInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	instance, err := s.repo.Update(c, request.ID, &models.Instance{
		ID: request.ID,
		Webhook: models.InstanceWebhook{
			Base64: &[]bool{request.Webhook.Base64}[0],
		},
	})
	if err != nil {
		if errors.Is(err, instances.ErrorNotFound) {
			return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
		}
		zap.L().Error("failed to create instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to update instance")
	}

	return ctx.JSON(http.StatusCreated, dto.UpdateInstanceResponse{
		Instance: instance,
	})
}

func (s *Instance) List(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.ListInstancesRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}
	if request.InstanceName == "" {
		request.InstanceName = request.ID
	}

	result, err := s.repo.List(c, request.InstanceName)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	var response []dto.ListInstancesResponse
	for _, instance := range result {
		jid, err := types.ParseJID(instance.RemoteJID)
		if err != nil {
			zap.L().Error("failed to parse jid", zap.Error(err))
		}

		// Buscar status da instância
		status, err := s.whatsmiau.Status(instance.ID)
		if err != nil {
			zap.L().Error("failed to get status for instance", zap.Error(err), zap.String("instance", instance.ID))
			status = "closed" // Status padrão se não conseguir obter
		}

		response = append(response, dto.ListInstancesResponse{
			Instance: &instance,
			OwnerJID: jid.ToNonAD().String(),
			Status:   string(status),
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

func (s *Instance) Connect(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.ConnectInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
	}

	qrCode, err := s.whatsmiau.Connect(c, request.ID)
	if err != nil {
		zap.L().Error("failed to connect instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to connect instance")
	}
	if qrCode != "" {
		png, err := qrcode.Encode(qrCode, qrcode.Medium, 256)
		if err != nil {
			zap.L().Error("failed to encode qrcode", zap.Error(err))
			return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to encode qrcode")
		}
		return ctx.JSON(http.StatusOK, dto.ConnectInstanceResponse{
			Message:   "If instance restart this instance could be lost if you cannot connect",
			Connected: false,
			Base64:    "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		})
	}

	return ctx.JSON(http.StatusOK, dto.ConnectInstanceResponse{
		Message:   "instance already connected",
		Connected: true,
	})
}

func (s *Instance) Status(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.ConnectInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
	}

	status, err := s.whatsmiau.Status(request.ID)
	if err != nil {
		zap.L().Error("failed to get status instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to get status instance")
	}

	// Buscar remoteJid se a instância estiver conectada
	var remoteJID string
	if string(status) == "open" && len(result) > 0 {
		remoteJID = result[0].RemoteJID
	}

	return ctx.JSON(http.StatusOK, dto.StatusInstanceResponse{
		ID:        request.ID,
		Status:    string(status),
		RemoteJID: remoteJID,
		Instance: &dto.StatusInstanceResponseEvolutionCompatibility{
			InstanceName: request.ID,
			State:        string(status),
		},
	})
}

func (s *Instance) Logout(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.DeleteInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
	}

	if err := s.whatsmiau.Logout(c, request.ID); err != nil {
		zap.L().Error("failed to logout instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to logout instance")
	}

	return ctx.JSON(http.StatusOK, dto.DeleteInstanceResponse{
		Message: "instance logout successfully",
	})
}

func (s *Instance) Delete(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.DeleteInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return ctx.JSON(http.StatusOK, dto.DeleteInstanceResponse{
			Message: "instance doesn't exists",
		})
	}

	if err := s.whatsmiau.Logout(c, request.ID); err != nil {
		zap.L().Error("failed to logout instance", zap.Error(err))
	}

	if err := s.whatsmiau.Disconnect(request.ID); err != nil {
		zap.L().Error("failed to disconnect instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to disconnect instance")
	}

	if err := s.repo.Delete(c, request.ID); err != nil {
		zap.L().Error("failed to delete instance", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to delete instance")
	}

	return ctx.JSON(http.StatusOK, dto.DeleteInstanceResponse{
		Message: "instance deleted",
	})
}

func (s *Instance) UpdateReadSettings(ctx echo.Context) error {
	var request dto.UpdateReadSettingsRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	instance, err := s.repo.Update(c, request.ID, &models.Instance{
		ID:               request.ID,
		AutoReadMessages: request.AutoReadMessages,
		ReadDelay:        request.ReadDelay,
	})
	if err != nil {
		if errors.Is(err, instances.ErrorNotFound) {
			return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
		}
		zap.L().Error("failed to update read settings", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to update read settings")
	}

	return ctx.JSON(http.StatusOK, dto.UpdateReadSettingsResponse{
		Instance: instance,
	})
}

// StartPairing inicia o processo de pairing para uma instância
func (s *Instance) StartPairing(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.StartPairingRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	// Verificar se a instância existe
	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
	}

	// Iniciar pairing
	response, err := s.whatsmiau.StartPairing(c, request.ID, request.PhoneNumber, request.ClientType, request.ClientName)
	if err != nil {
		zap.L().Error("failed to start pairing", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to start pairing")
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetPairingStatus retorna o status atual do pairing
func (s *Instance) GetPairingStatus(ctx echo.Context) error {
	c := ctx.Request().Context()
	var request dto.PairingStatusRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	// Verificar se a instância existe
	result, err := s.repo.List(c, request.ID)
	if err != nil {
		zap.L().Error("failed to list instances", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to list instances")
	}

	if len(result) == 0 {
		return utils.HTTPFail(ctx, http.StatusNotFound, err, "instance not found")
	}

	// Obter status do pairing
	response, err := s.whatsmiau.GetPairingStatus(c, request.ID, request.SessionID)
	if err != nil {
		zap.L().Error("failed to get pairing status", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to get pairing status")
	}

	return ctx.JSON(http.StatusOK, response)
}

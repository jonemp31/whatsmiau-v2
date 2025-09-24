package controllers

import (
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/verbeux-ai/whatsmiau/interfaces"
	"github.com/verbeux-ai/whatsmiau/lib/whatsmiau"
	"github.com/verbeux-ai/whatsmiau/server/dto"
	"github.com/verbeux-ai/whatsmiau/utils"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

type Message struct {
	repo      interfaces.InstanceRepository
	whatsmiau *whatsmiau.Whatsmiau
}

func NewMessages(repository interfaces.InstanceRepository, whatsmiau *whatsmiau.Whatsmiau) *Message {
	return &Message{
		repo:      repository,
		whatsmiau: whatsmiau,
	}
}

func (s *Message) SendText(ctx echo.Context) error {
	var request dto.SendTextRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	jid, err := numberToJid(request.Number)
	if err != nil {
		zap.L().Error("error converting number to jid", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid number format")
	}

	sendText := &whatsmiau.SendText{
		Text:       request.Text,
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
	}

	if request.Quoted != nil && len(request.Quoted.Key.Id) > 0 && len(request.Quoted.Message.Conversation) > 0 {
		sendText.QuoteMessage = request.Quoted.Message.Conversation
		sendText.QuoteMessageID = request.Quoted.Key.Id
	}

	c := ctx.Request().Context()
	if err := s.whatsmiau.ChatPresence(&whatsmiau.ChatPresenceRequest{
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
		Presence:   types.ChatPresenceComposing,
	}); err != nil {
		zap.L().Error("Whatsmiau.ChatPresence", zap.Error(err))
	} else {
		// Delay padrão de 6 segundos para texto
		delay := request.Delay
		if delay == 0 {
			delay = 6000 // 6 segundos em millisegundos
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}

	res, err := s.whatsmiau.SendText(c, sendText)
	if err != nil {
		zap.L().Error("Whatsmiau.SendText failed", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send text")
	}

	return ctx.JSON(http.StatusOK, dto.SendTextResponse{
		Key: dto.MessageResponseKey{
			RemoteJid: request.Number,
			FromMe:    true,
			Id:        res.ID,
		},
		Status: "sent",
		Message: dto.SendTextResponseMessage{
			Conversation: request.Text,
		},
		MessageType:      "conversation",
		MessageTimestamp: int(res.CreatedAt.Unix() / 1000),
		InstanceId:       request.InstanceID,
	})
}

func (s *Message) SendAudio(ctx echo.Context) error {
	var request dto.SendAudioRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	jid, err := numberToJid(request.Number)
	if err != nil {
		zap.L().Error("error converting number to jid", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid number format")
	}

	sendText := &whatsmiau.SendAudio{
		AudioURL:   request.Audio,
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
		ViewOnce:   request.ViewOnce,
	}

	if request.Quoted != nil && len(request.Quoted.Key.Id) > 0 && len(request.Quoted.Message.Conversation) > 0 {
		sendText.QuoteMessage = request.Quoted.Message.Conversation
		sendText.QuoteMessageID = request.Quoted.Key.Id
	}

	c := ctx.Request().Context()
	if err := s.whatsmiau.ChatPresence(&whatsmiau.ChatPresenceRequest{
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
		Presence:   types.ChatPresenceComposing,
		Media:      types.ChatPresenceMediaAudio,
	}); err != nil {
		zap.L().Error("Whatsmiau.ChatPresence", zap.Error(err))
	} else {
		// Delay padrão de 12 segundos para áudio
		delay := request.Delay
		if delay == 0 {
			delay = 12000 // 12 segundos em millisegundos
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}

	res, err := s.whatsmiau.SendAudio(c, sendText)
	if err != nil {
		zap.L().Error("Whatsmiau.SendAudio failed", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send audio")
	}

	return ctx.JSON(http.StatusOK, dto.SendAudioResponse{
		Key: dto.MessageResponseKey{
			RemoteJid: request.Number,
			FromMe:    true,
			Id:        res.ID,
		},

		Status:           "sent",
		MessageType:      "audioMessage",
		MessageTimestamp: int(res.CreatedAt.Unix() / 1000),
		InstanceId:       request.InstanceID,
	})
}

// For evolution compatibility
func (s *Message) SendMedia(ctx echo.Context) error {
	var request dto.SendMediaRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}
	switch request.Mediatype {
	case "image":
		request.SendDocumentRequest.Mimetype = "image/png"
		return s.sendImage(ctx, request.SendDocumentRequest)
	case "video":
		return s.sendVideo(ctx, request.SendDocumentRequest)
	}

	return s.sendDocument(ctx, request.SendDocumentRequest)
}

func (s *Message) SendDocument(ctx echo.Context) error {
	var request dto.SendDocumentRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	return s.sendDocument(ctx, request)
}

func (s *Message) sendDocument(ctx echo.Context, request dto.SendDocumentRequest) error {
	jid, err := numberToJid(request.Number)
	if err != nil {
		zap.L().Error("error converting number to jid", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid number format")
	}

	sendData := &whatsmiau.SendDocumentRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Caption:    request.Caption,
		FileName:   request.FileName,
		RemoteJID:  jid,
		Mimetype:   request.Mimetype,
	}

	c := ctx.Request().Context()
	time.Sleep(time.Millisecond * time.Duration(request.Delay)) // TODO: create a more robust solution

	res, err := s.whatsmiau.SendDocument(c, sendData)
	if err != nil {
		zap.L().Error("Whatsmiau.SendDocument failed", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send document")
	}

	return ctx.JSON(http.StatusOK, dto.SendDocumentResponse{
		Key: dto.MessageResponseKey{
			RemoteJid: request.Number,
			FromMe:    true,
			Id:        res.ID,
		},
		Status:           "sent",
		MessageType:      "documentMessage",
		MessageTimestamp: int(res.CreatedAt.Unix() / 1000),
		InstanceId:       request.InstanceID,
	})
}

func (s *Message) SendImage(ctx echo.Context) error {
	var request dto.SendDocumentRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	return s.sendImage(ctx, request)
}

func (s *Message) sendImage(ctx echo.Context, request dto.SendDocumentRequest) error {
	jid, err := numberToJid(request.Number)
	if err != nil {
		zap.L().Error("error converting number to jid", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid number format")
	}

	sendData := &whatsmiau.SendImageRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Caption:    request.Caption,
		RemoteJID:  jid,
		Mimetype:   request.Mimetype,
		ViewOnce:   request.ViewOnce,
	}

	c := ctx.Request().Context()
	if err := s.whatsmiau.ChatPresence(&whatsmiau.ChatPresenceRequest{
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
		Presence:   types.ChatPresenceComposing,
		Media:      types.ChatPresenceMediaText,
	}); err != nil {
		zap.L().Error("Whatsmiau.ChatPresence", zap.Error(err))
	} else {
		// Delay padrão de 2 segundos para imagem
		delay := request.Delay
		if delay == 0 {
			delay = 2000 // 2 segundos em millisegundos
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}

	res, err := s.whatsmiau.SendImage(c, sendData)
	if err != nil {
		zap.L().Error("Whatsmiau.SendDocument failed", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send document")
	}

	return ctx.JSON(http.StatusOK, dto.SendDocumentResponse{
		Key: dto.MessageResponseKey{
			RemoteJid: request.Number,
			FromMe:    true,
			Id:        res.ID,
		},
		Status:           "sent",
		MessageType:      "imageMessage",
		MessageTimestamp: int(res.CreatedAt.Unix() / 1000),
		InstanceId:       request.InstanceID,
	})
}

func (s *Message) SendVideo(ctx echo.Context) error {
	var request dto.SendDocumentRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	return s.sendVideo(ctx, request)
}

func (s *Message) sendVideo(ctx echo.Context, request dto.SendDocumentRequest) error {
	jid, err := numberToJid(request.Number)
	if err != nil {
		zap.L().Error("error converting number to jid", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid number format")
	}

	sendData := &whatsmiau.SendVideoRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Caption:    request.Caption,
		RemoteJID:  jid,
		Mimetype:   request.Mimetype,
		ViewOnce:   request.ViewOnce,
	}

	c := ctx.Request().Context()
	if err := s.whatsmiau.ChatPresence(&whatsmiau.ChatPresenceRequest{
		InstanceID: request.InstanceID,
		RemoteJID:  jid,
		Presence:   types.ChatPresenceComposing,
		Media:      types.ChatPresenceMediaText,
	}); err != nil {
		zap.L().Error("Whatsmiau.ChatPresence", zap.Error(err))
	} else {
		// Delay padrão de 2 segundos para vídeo
		delay := request.Delay
		if delay == 0 {
			delay = 2000 // 2 segundos em millisegundos
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}

	res, err := s.whatsmiau.SendVideo(c, sendData)
	if err != nil {
		zap.L().Error("Whatsmiau.SendVideo failed", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send video")
	}

	return ctx.JSON(http.StatusOK, dto.SendDocumentResponse{
		Key: dto.MessageResponseKey{
			RemoteJid: request.Number,
			FromMe:    true,
			Id:        res.ID,
		},
		Status:           "sent",
		MessageType:      "videoMessage",
		MessageTimestamp: int(res.CreatedAt.Unix() / 1000),
		InstanceId:       request.InstanceID,
	})
}

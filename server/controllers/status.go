package controllers

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/verbeux-ai/whatsmiau/lib/whatsmiau"
	"github.com/verbeux-ai/whatsmiau/server/dto"
	"github.com/verbeux-ai/whatsmiau/utils"
	"go.uber.org/zap"
)

type Status struct {
	whatsmiau *whatsmiau.Whatsmiau
}

func NewStatus(whatsmiau *whatsmiau.Whatsmiau) *Status {
	return &Status{whatsmiau: whatsmiau}
}

func (s *Status) SendText(ctx echo.Context) error {
	var request dto.SendStatusTextRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	res, err := s.whatsmiau.SendStatusText(c, &whatsmiau.SendStatusTextRequest{
		InstanceID: request.InstanceID,
		Text:       request.Text,
		Background: request.Background,
		Font:       request.Font,
	})

	if err != nil {
		zap.L().Error("failed to send status text", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send status text")
	}

	return ctx.JSON(http.StatusOK, dto.SendStatusResponse{
		ID:          res.ID,
		CreatedAt:   res.CreatedAt.Unix(),
		InstanceID:  request.InstanceID,
		MessageType: "text",
		Status:      "sent",
	})
}

func (s *Status) SendImage(ctx echo.Context) error {
	var request dto.SendStatusMediaRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	res, err := s.whatsmiau.SendStatusImage(c, &whatsmiau.SendStatusImageRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Caption:    request.Caption,
		Mimetype:   request.Mimetype,
	})

	if err != nil {
		zap.L().Error("failed to send status image", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send status image")
	}

	return ctx.JSON(http.StatusOK, dto.SendStatusResponse{
		ID:          res.ID,
		CreatedAt:   res.CreatedAt.Unix(),
		InstanceID:  request.InstanceID,
		MessageType: "image",
		Status:      "sent",
	})
}

func (s *Status) SendVideo(ctx echo.Context) error {
	var request dto.SendStatusMediaRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	res, err := s.whatsmiau.SendStatusVideo(c, &whatsmiau.SendStatusVideoRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Caption:    request.Caption,
		Mimetype:   request.Mimetype,
	})

	if err != nil {
		zap.L().Error("failed to send status video", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send status video")
	}

	return ctx.JSON(http.StatusOK, dto.SendStatusResponse{
		ID:          res.ID,
		CreatedAt:   res.CreatedAt.Unix(),
		InstanceID:  request.InstanceID,
		MessageType: "video",
		Status:      "sent",
	})
}

func (s *Status) SendAudio(ctx echo.Context) error {
	var request dto.SendStatusMediaRequest
	if err := ctx.Bind(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusUnprocessableEntity, err, "failed to bind request body")
	}

	if err := validator.New().Struct(&request); err != nil {
		return utils.HTTPFail(ctx, http.StatusBadRequest, err, "invalid request body")
	}

	c := ctx.Request().Context()
	res, err := s.whatsmiau.SendStatusAudio(c, &whatsmiau.SendStatusAudioRequest{
		InstanceID: request.InstanceID,
		MediaURL:   request.Media,
		Mimetype:   request.Mimetype,
	})

	if err != nil {
		zap.L().Error("failed to send status audio", zap.Error(err))
		return utils.HTTPFail(ctx, http.StatusInternalServerError, err, "failed to send status audio")
	}

	return ctx.JSON(http.StatusOK, dto.SendStatusResponse{
		ID:          res.ID,
		CreatedAt:   res.CreatedAt.Unix(),
		InstanceID:  request.InstanceID,
		MessageType: "audio",
		Status:      "sent",
	})
}

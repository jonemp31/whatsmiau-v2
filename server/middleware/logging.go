package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// LoggingMiddleware cria logs estruturados para requisições HTTP
func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Processar requisição
			err := next(c)

			// Calcular duração
			duration := time.Since(start)

			// Log da requisição
			zap.L().Info("HTTP Request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.String("remote_ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.Int("status", c.Response().Status),
				zap.Duration("duration", duration),
				zap.String("instance", c.Param("instance")),
			)

			return err
		}
	}
}

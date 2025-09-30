package logger

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// StartLogger inicializa o sistema de logs centralizado
func StartLogger() error {
	// Criar diretório de logs
	os.MkdirAll("logs", 0755)

	// Configuração do console (desenvolvimento)
	consoleConfig := zap.NewDevelopmentConfig()
	consoleLogger, _ := consoleConfig.Build()

	// Lumberjack para rotação diária
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "logs/whatsmiau.log",
		MaxSize:    0,     // Sem limite de tamanho
		MaxBackups: 1,     // Manter apenas 1 backup
		MaxAge:     1,     // Manter por 1 dia (24h + 12h = 36h)
		Compress:   false, // Não comprimir
		LocalTime:  true,  // Usar horário local
	}

	// Encoder para arquivo (JSON estruturado)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// Core do arquivo
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(lumberjackLogger),
		zap.InfoLevel,
	)

	// Combinar console + arquivo
	logger := zap.New(zapcore.NewTee(
		consoleLogger.Core(),
		fileCore,
	))

	// Substituir o logger global
	zap.ReplaceGlobals(logger)

	// Iniciar limpeza automática
	go startLogCleanup()

	return nil
}

// startLogCleanup inicia a limpeza automática de logs antigos
func startLogCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // Verificar a cada hora
	defer ticker.Stop()

	for range ticker.C {
		cleanOldLogs()
	}
}

// cleanOldLogs remove logs mais antigos que 36 horas
func cleanOldLogs() {
	logDir := "logs"
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-36 * time.Hour)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Verificar se é arquivo de log
		if filepath.Ext(file.Name()) == ".log" {
			info, err := file.Info()
			if err != nil {
				continue
			}

			// Remover arquivos mais antigos que 36 horas
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(logDir, file.Name()))
			}
		}
	}
}

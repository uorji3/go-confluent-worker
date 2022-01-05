package logger

import (
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
}

func newZapLogger() *zapLogger {
	if os.Getenv("DISABLE_STDOUT_LOGGER") == "true" {
		return &zapLogger{}
	}

	level := zapcore.InfoLevel
	if os.Getenv("ENVIRONMENT") == "development" {
		level = zapcore.DebugLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.Sampling = nil
	cfg.EncoderConfig = encoderConfig()

	logger, err := cfg.Build()
	if err != nil {
		log.Fatalf("while initializing zap logger: %v", err)
	}

	return &zapLogger{
		logger: logger,
	}
}

func (l *zapLogger) Panic(msg string) {
	l.logger.Panic(msg)
}

func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panic(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Fatal(msg string) {
	l.logger.Fatal(msg)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Info(msg string) {
	l.logger.Info(msg)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *zapLogger) Flush() error {
	if l.logger != nil {
		return l.logger.Sync()
	}

	return nil
}

func (l *zapLogger) Valid() bool {
	return l.logger != nil
}

func encoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()

	return zapcore.EncoderConfig{
		MessageKey:   "message",
		LevelKey:     "severity",
		TimeKey:      "time",
		NameKey:      cfg.NameKey,
		LineEnding:   cfg.LineEnding,
		EncodeLevel:  cfg.EncodeLevel,
		EncodeTime:   zapcore.RFC3339NanoTimeEncoder,
		EncodeCaller: cfg.EncodeCaller,
		EncodeName:   cfg.EncodeName,
	}
}

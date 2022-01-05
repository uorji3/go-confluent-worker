package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/logging"
	"google.golang.org/api/option"
)

type gcpLogger struct {
	client *logging.Client
	logger *logging.Logger
}

func newGcpLogger() *gcpLogger {
	if os.Getenv("ENABLE_GCP_LOGGER") != "true" {
		return &gcpLogger{}
	}

	credsStr := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsStr == "" {
		return &gcpLogger{}
	}

	b := []byte(credsStr)
	opts := option.WithCredentialsJSON(b)
	var creds map[string]interface{}
	err := json.Unmarshal(b, &creds)
	if err != nil {
		log.Fatal(err)
	}

	projectID := creds["project_id"].(string)
	ctx := context.Background()
	client, err := logging.NewClient(ctx, projectID, opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	loggerName := "confluent-metrics-worker"

	gcpLoggerName := os.Getenv("GCP_LOGGER_NAME")
	if gcpLoggerName != "" {
		loggerName = gcpLoggerName
	}

	logger := client.Logger(loggerName)

	return &gcpLogger{
		client: client,
		logger: logger,
	}
}

func (l *gcpLogger) Panic(msg string) {
	l.logMessage(msg, logging.Error)
}

func (l *gcpLogger) Panicf(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Error)
}

func (l *gcpLogger) Fatal(msg string) {
	l.logMessage(msg, logging.Error)
}

func (l *gcpLogger) Fatalf(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Error)
}

func (l *gcpLogger) Error(msg string) {
	l.logMessage(msg, logging.Error)
}

func (l *gcpLogger) Errorf(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Error)
}

func (l *gcpLogger) Warn(msg string) {
	l.logMessage(msg, logging.Warning)
}

func (l *gcpLogger) Warnf(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Warning)
}

func (l *gcpLogger) Info(msg string) {
	l.logMessage(msg, logging.Info)
}

func (l *gcpLogger) Infof(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Info)
}

func (l *gcpLogger) Debug(msg string) {
	l.logMessage(msg, logging.Debug)
}

func (l *gcpLogger) Debugf(format string, args ...interface{}) {
	l.logMessage(fmt.Sprintf(format, args...), logging.Debug)
}

func (l *gcpLogger) Flush() error {
	if l.logger != nil {
		l.logger.Flush()
	}

	if l.client != nil {
		l.client.Close()
	}

	return nil
}

func (l *gcpLogger) Valid() bool {
	return l.logger != nil
}

func (l *gcpLogger) logMessage(payload string, severity logging.Severity) {
	l.logger.Log(logging.Entry{
		Payload:  payload,
		Severity: severity,
	})
}

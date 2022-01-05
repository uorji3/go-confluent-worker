package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

type sentryLogger struct {
}

func newSentryLogger() *sentryLogger {
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:         sentryDSN,
			Environment: "production",
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
	}

	return &sentryLogger{}
}

func (l *sentryLogger) Panic(msg string) {
	sentry.CaptureMessage(msg)
}

func (l *sentryLogger) Panicf(format string, args ...interface{}) {
	sentry.CaptureMessage(fmt.Sprintf(format, args...))
}

func (l *sentryLogger) Fatal(msg string) {
	sentry.CaptureMessage(msg)
}

func (l *sentryLogger) Fatalf(format string, args ...interface{}) {
	sentry.CaptureMessage(fmt.Sprintf(format, args...))
}

func (l *sentryLogger) Error(msg string) {
	sentry.CaptureMessage(msg)
}

func (l *sentryLogger) Errorf(format string, args ...interface{}) {
	sentry.CaptureMessage(fmt.Sprintf(format, args...))
}

func (l *sentryLogger) Warn(msg string) {
}

func (l *sentryLogger) Warnf(format string, args ...interface{}) {
}

func (l *sentryLogger) Info(msg string) {
}

func (l *sentryLogger) Infof(format string, args ...interface{}) {
}

func (l *sentryLogger) Debug(msg string) {
}

func (l *sentryLogger) Debugf(format string, args ...interface{}) {
}

func (l *sentryLogger) Flush() error {
	sentry.Flush(2 * time.Second)
	return nil
}

func (l *sentryLogger) Valid() bool {
	return true
}

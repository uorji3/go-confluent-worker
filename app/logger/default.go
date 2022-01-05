package logger

import (
	"sync"
)

var (
	logger     = make([]MessageLogger, 0)
	loggerSync = &sync.Once{}
)

func init() {
	logger = Default()
}

func Initialize() {
	loggerSync = &sync.Once{}
	logger = Default()
}

func Default() []MessageLogger {
	return setupLogger()
}

func setupLogger() []MessageLogger {
	loggerSync.Do(func() {
		logger = make([]MessageLogger, 0)

		allMessageLoggers := []MessageLogger{
			newZapLogger(),
			newGcpLogger(),
			newSentryLogger(),
		}

		for _, messageLogger := range allMessageLoggers {
			if messageLogger.Valid() {
				logger = append(logger, messageLogger)
			}
		}
	})

	return logger
}

func Panic(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Panic(msg)
	}
}

func Panicf(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Panicf(format, args...)
	}
}

func Fatal(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Fatal(msg)
	}
}

func Fatalf(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Fatalf(format, args...)
	}
}

func Error(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Error(msg)
	}
}

func Errorf(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Errorf(format, args...)
	}
}

func Warn(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Warn(msg)
	}
}

func Warnf(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Warnf(format, args...)
	}
}

func Info(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Info(msg)
	}
}

func Infof(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Infof(format, args...)
	}
}

func Debug(msg string) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Debug(msg)
	}
}

func Debugf(format string, args ...interface{}) {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Debugf(format, args...)
	}
}

func Flush() error {
	msgLoggers := Default()
	for _, msgLogger := range msgLoggers {
		msgLogger.Flush()
	}

	return nil
}

package logger

import (
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"os"
)

type DTLogger struct {
	infoLogger  logr.Logger
	errorLogger logr.Logger
}

func NewDTLogger() logr.Logger {
	return DTLogger{
		infoLogger:  zap.LoggerTo(os.Stdout),
		errorLogger: zap.LoggerTo(os.Stderr),
	}
}

func (dtl DTLogger) Info(msg string, keysAndValues ...interface{}) {
	dtl.infoLogger.Info(msg, keysAndValues...)
}

func (dtl DTLogger) Enabled() bool {
	return dtl.infoLogger.Enabled()
}

func (dtl DTLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	dtl.errorLogger.Error(err, msg, keysAndValues...)
}

func (dtl DTLogger) V(level int) logr.InfoLogger {
	return dtl.errorLogger.V(level)
}

func (dtl DTLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return DTLogger{
		infoLogger:  dtl.infoLogger.WithValues(keysAndValues...),
		errorLogger: dtl.errorLogger.WithValues(keysAndValues...),
	}
}

func (dtl DTLogger) WithName(name string) logr.Logger {
	return DTLogger{
		infoLogger:  dtl.infoLogger.WithName(name),
		errorLogger: dtl.errorLogger.WithName(name),
	}
}

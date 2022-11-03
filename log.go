package main

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initLogging() (logr.Logger, *zap.Logger) {
	zc := zap.NewProductionConfig()
	zc.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLog, err := zc.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to setup logging: %v", err))
	}

	log := zapr.NewLogger(zapLog)
	zap.RedirectStdLog(zapLog)
	zap.ReplaceGlobals(zapLog)
	return log, zapLog
}

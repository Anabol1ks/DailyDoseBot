package logger

import (
	"go.uber.org/zap"
)

var log *zap.Logger

func Init() error {
	var err error
	log, err = zap.NewDevelopment()
	if err != nil {
		return err
	}
	return nil
}

func L() *zap.Logger {
	if log == nil {
		panic("Logger not initialized")
	}
	return log
}

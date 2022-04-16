package logger

import "github.com/go-kit/kit/log"

type Logger interface {
	SetLoglevelFromConfig(cfg loglevelConfig)
	SetLogLevel(level string)
	Error(params ...interface{})
	Warn(params ...interface{})
	Info(params ...interface{})
	Notice(params ...interface{})
	Debug(params ...interface{})
	Fatal(params ...interface{})
	With(key string, value interface{}) Logger
	WithPrefix(prefix string) Logger
	WithErrorCounter(metric incrementable) Logger
	GetGoKitLogger() log.Logger
	// Deprecated: Don't use this function directly! It's meant to only be used by go-kit logger
	Log(keyvals ...interface{}) error
	logError(err error, lvl level, depth int)
}

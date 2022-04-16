package levelfilter

import (
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	levelKey = "severity"
)

// https://github.com/go-kit/kit/issues/503
func setLevelKey(logger log.Logger, key interface{}) log.Logger {
	return log.LoggerFunc(func(keyvals ...interface{}) error {
		tsKeyIndex := -1
		hasTime := false

		for i := 1; i < len(keyvals); i += 2 {

			// For stackdriver
			if keyvals[i-1] == "msg" {
				keyvals[i-1] = "message"
			}

			if keyvals[i-1] == "ts" {
				tsKeyIndex = i - 1
			}

			if keyvals[i-1] == "time" {
				hasTime = true
			}

			if _, ok := keyvals[i].(level.Value); ok {
				// overwriting the key without copying keyvals
				// techically violates the log.Logger contract
				// but is safe in this context because none
				// of the loggers in this program retain a reference
				// to keyvals
				keyvals[i-1] = key
			}
		}

		if !hasTime && tsKeyIndex != -1 {
			keyvals[tsKeyIndex] = "time"
		}

		return logger.Log(keyvals...)
	})
}

func wrapErrors(logger log.Logger) log.Logger {
	return log.LoggerFunc(func(keyvals ...interface{}) error {
		if err := logger.Log(keyvals...); err != nil {
			println("log error:", err)
		}
		return nil
	})
}

func Injector(lvl string, logger log.Logger) log.Logger {
	switch strings.ToLower(lvl) {
	case "debug":
		return level.NewInjector(logger, level.DebugValue())
	case "info":
		return level.NewInjector(logger, level.InfoValue())
	case "warn":
		return level.NewInjector(logger, level.WarnValue())
	case "warning":
		return level.NewInjector(logger, level.WarnValue())
	case "error":
		return level.NewInjector(logger, level.ErrorValue())
	case "fatal":
		return level.NewInjector(logger, level.ErrorValue())
	default:
		return level.NewInjector(logger, level.InfoValue())
	}
}

// Filter creates a new go-kit/log filter
func Filter(lvl string, logger log.Logger) log.Logger {
	logger = setLevelKey(logger, levelKey)
	logger = wrapErrors(logger)
	// set the allowed log level filter
	switch strings.ToLower(lvl) {
	case "debug":
		return level.NewFilter(logger, level.AllowDebug())
	case "info":
		return level.NewFilter(logger, level.AllowInfo())
	case "warn":
		return level.NewFilter(logger, level.AllowWarn())
	case "warning":
		return level.NewFilter(logger, level.AllowWarn())
	case "error":
		return level.NewFilter(logger, level.AllowError())
	case "fatal":
		return level.NewFilter(logger, level.AllowError())
	default:
		return level.NewFilter(logger, level.AllowInfo())
	}
}

// FromEnv creates a new go-kit/log/level filter reading LOGLEVEL
func FromEnv(logger log.Logger) log.Logger {
	return Filter(os.Getenv("LOGLEVEL"), logger)
}

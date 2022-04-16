package logger

import (
	"fmt"
	"io"

	"os"

	"strings"

	"runtime"

	"strconv"

	"path/filepath"

	"github.com/fatih/color"
	"github.com/go-kit/kit/log"
	gklevel "github.com/go-kit/kit/log/level"
	"github.com/josephmbassey/calculator-service/internals/http/handler/version"
	"github.com/josephmbassey/calculator-service/internals/logger/levelfilter"
	"github.com/pkg/errors"
)

// level is the data format the loglevels are saved as
type level byte

// incrementable is simply anything that we can call Inc() on (usually various prometheus collectors)
type incrementable interface {
	Inc()
}

// GKLogger is the gokit logger. It is aliased here to make it easier to reference externally by avoiding
// package aliases
type GKLogger = log.Logger

// These are the different loglevels
const (
	levelDebug level = 1 << iota
	levelInfo
	levelNotice
	levelWarn
	levelError
	levelCritical
)

// allLevelsFrom enables all levels above the provided one, including the provided one
func allLevelsFrom(loglevel level) level {
	return ^(loglevel - 1)
}

// excludes only debug
const levelDefault = ^(levelInfo - 1)

// ToGokitFormat tries to get the gokit level version. However due to the nature of this format, it can only
// use a heuristic
func (l level) ToGokitFormat() string {
	if l&levelDebug != 0 {
		return "debug"
	}
	if l&levelInfo != 0 {
		return "info"
	}
	if l&levelNotice != 0 {
		return "info"
	}
	if l&levelWarn != 0 {
		return "warn"
	}
	return "error"
}

// String returns the enum as a string
func (l level) String() string {
	var retParams []string

	if l&levelDebug != 0 {
		retParams = append(retParams, "debug")
	}
	if l&levelInfo != 0 {
		retParams = append(retParams, "info")
	}
	if l&levelNotice != 0 {
		retParams = append(retParams, "notice")
	}
	if l&levelWarn != 0 {
		retParams = append(retParams, "warn")
	}
	if l&levelError != 0 {
		retParams = append(retParams, "error")
	}
	if l&levelCritical != 0 {
		retParams = append(retParams, "critical")
	}

	return strings.Join(retParams, " | ")
}

// logger is a wrapper around the GoKit logger with extended features
type logger struct {
	gklogger        log.Logger
	loglevel        level
	errorCounter    incrementable
	prefix          string
	useLegacyLevels bool
	useCustomLogger bool
}

// New creates a new logger from the provided environment
func New(env string) Logger {
	return NewWithFormat(env, "")
}

// New creates a new logger from the provided environment or optionally a custom logger for dev
func NewWithFormat(env string, format string) Logger {
	var gklogger log.Logger

	writer := os.Stdout

	var useCustomLogger bool
	var useColors bool

	if os.Getenv("LOGPLAIN") == "" {
		useColors = true
	}

	switch format {
	// fallback format = JSON format
	case "", "json":
		gklogger = log.NewJSONLogger(writer)
	default:
		if format == "quick" {
			format = "[$severity_s][$source_location] $message $."
		}
		useCustomLogger = true
		gklogger = &customFormatLogger{
			fmtstring: format,
			useColors: useColors,
			writer:    writer,
		}
	}

	gklogger = log.With(gklogger,
		"time", log.DefaultTimestampUTC,
		"name", version.Name,
		"commit", version.Commit,
	)

	l := createLogger(gklogger)

	if useCustomLogger {
		l.useCustomLogger = true
		color.NoColor = false
	}

	return l
}

func createTestLogger(w io.Writer) *logger {
	gklogger := log.NewJSONLogger(w)

	return createLogger(gklogger)
}

func createLogger(gklogger log.Logger) *logger {
	return &logger{
		gklogger: gklogger,
		loglevel: levelDefault,
	}
}

// NewFromEnv figures out the environment on its own.
func NewFromEnv() Logger {
	return NewWithFormat(os.Getenv("ENVIRONMENT"), os.Getenv("LOGFMT"))
}

type loglevelConfig interface {
	GetLoglevel() string
}

// Embed LoglevelEnv in your Config to quickly add configurable loglevel
type LoglevelEnv struct {
	Loglevel string `arg:"--loglevel,env:LOGLEVEL"`
}

func (lvl LoglevelEnv) GetLoglevel() string {
	return lvl.Loglevel
}

func (l *logger) SetLoglevelFromConfig(cfg loglevelConfig) {
	l.SetLogLevel(cfg.GetLoglevel())
}

func (l *logger) SetLogLevel(level string) {
	lvl, err := allLoglevelsFromString(level)
	if err != nil {
		l.Error(err)
	}
	l.loglevel = lvl
}

// loglevelFromString gets the loglevel from environment variables / cli params
func loglevelFromString(loglevel string) (level, error) {
	switch strings.ToLower(loglevel) {
	case "debug":
		return levelDebug, nil
	case "info":
		return levelInfo, nil
	case "notice":
		return levelNotice, nil
	case "warn", "warning":
		return levelWarn, nil
	case "error", "err":
		return levelError, nil
	case "fatal", "critical":
		return levelCritical, nil
	case "":
		return levelDebug, nil
	default:
		return levelDebug, errors.Errorf("loglevel is set to unknown value `%s`", loglevel)
	}
}

// allLoglevelsFromString gets all the according loglevels from the basic level
func allLoglevelsFromString(loglevel string) (level, error) {
	l, err := loglevelFromString(loglevel)
	return allLevelsFrom(l), err
}

type sourceLocation struct {
	File     string `json:"file"`
	Line     string `json:"line"`
	Function string `json:"function"`
}

func (s sourceLocation) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

func (s sourceLocation) String() string {
	return fmt.Sprintf("%s @ %s:%s", stripPackagePath(s.Function), s.File, s.Line)
}

func (s sourceLocation) MarshalJSON() (text []byte, err error) {
	return []byte(fmt.Sprintf(`{"file":"%s","line":%s,"function":"%s"}`, s.File, s.Line, s.Function)), nil
}

func stripPackagePath(fn string) string {
	p := fn[strings.LastIndexByte(fn, '/')+1:]
	return p[strings.IndexRune(p, '.')+1:]
}

// logWithFallback is the function that logs the call or falls back on println in case there's some problem.
func (l logger) logWithFallback(lvl level, depth int, params ...interface{}) {
	if !l.isAllowed(lvl) {
		return
	}

	hasSourceLocation := false

	for i := range params {
		// We only care about the keys
		if i%2 == 1 {
			continue
		}

		switch params[i] {
		case "message":
			msg := fmt.Sprint(params[i+1])

			if l.prefix != "" && len(params) > i+1 {
				msg = "[" + l.prefix + "] " + msg
				//if l.useCustomLogger {
				//	msg = useColorForLevel(lvl, msg)
				//}
				params[i+1] = msg
			} else {
				//if l.useCustomLogger {
				//	params[i+1] = useColorForLevel(lvl, msg)
				//}
			}
		case "source_location":
			hasSourceLocation = true
		}
	}

	var lg log.Logger

	if l.useLegacyLevels {
		var fn func(logger log.Logger) log.Logger

		switch lvl {
		case levelDebug:
			fn = gklevel.Debug
		case levelInfo:
			fn = gklevel.Info
		case levelNotice:
			fn = gklevel.Info
		case levelWarn:
			fn = gklevel.Warn
		case levelError:
			fn = gklevel.Error
		case levelCritical:
			fn = gklevel.Error
		}

		lg = fn(l.gklogger)
	} else {
		lg = log.With(l.gklogger, "severity", lvl.String())
	}

	pc, path, lineNo, ok := runtime.Caller(depth + 2)
	details := runtime.FuncForPC(pc)

	if l.useCustomLogger {

	}
	wd, _ := os.Getwd()
	relPath, err := filepath.Rel(wd, path)
	if err == nil {
		path = relPath
	}

	if ok && details != nil && !hasSourceLocation {
		params = append(params, "source_location", sourceLocation{
			File:     path,
			Line:     strconv.Itoa(lineNo),
			Function: details.Name(),
		})
	}

	if err := lg.Log(params...); err != nil {
		println("Error when trying to log message: " + err.Error())
		println("Falling back to println")
		var paramStr = fmt.Sprintf("\tseverity: %s\n", lvl.String())
		for i := 0; i < len(params); i += 2 {
			paramStr += fmt.Sprintf("%v: %v\n", params[i], params[i+1])
		}
		println(fmt.Sprintf("<<<BEGIN LOG\n%s\nEND LOG>>>", paramStr))
	}
}

// isAllowed checks whether we're filtering the messages on this level
func (l logger) isAllowed(lvl level) bool {
	return (l.loglevel & lvl) != 0
}

// logError is a helper function that logs an error with a stacktrace
func (l logger) logError(err error, lvl level, depth int) {
	st := getUnderlyingStacktrace(err)

	l.logWithFallback(lvl, depth+1, "message", err.Error(), "stacktrace", stacktrace{st: st})
}

// logFancy parses the multiple params to recognize how the log parameters should look like.
func (l logger) logFancy(depth int, lvl level, p ...interface{}) {
	depth++

	if lvl == levelCritical || lvl == levelError {
		l.incMetric()
	}

	if len(p) == 0 {
		if lvl == levelCritical || lvl == levelError {
			l.logError(errors.New("No Error message has been provided"), lvl, depth)
		} else {
			l.logWithFallback(lvl, depth, "message", "No message has been provided")
		}
		return
	}

	params := p[1:]
	textOrErr := p[0]

	if textOrErr == nil {
		l.logError(errors.New("error is nil"), levelCritical, depth)
		return
	}

	if err, ok := textOrErr.(error); ok {
		if err == nil {
			l.logError(errors.New("error is nil"), levelCritical, depth)
			return
		}
		if _, ok := err.(stackTracer); !ok {
			err = ErrWithStacktrace(err, depth)
		}
		if len(params) >= 1 {
			str, ok := params[0].(string)
			if !ok {
				l.logError(errors.New("second parameter to logger needs to be a string"), levelCritical, depth)
				return
			}

			text := fmt.Sprintf(str, params[1:]...)

			l.logError(errors.Wrap(err, text), lvl, depth)
		} else {
			l.logError(err, lvl, depth)
		}
	} else if text, ok := textOrErr.(string); ok {
		if len(params) == 1 && !strings.Contains(text, "%") {
			if err, ok := params[0].(error); ok {
				// most likely the parameter ordering is the wrong way around.
				l.logError(errors.New("parameter order in the logger should be (error, string), not (string, error)"), levelCritical, depth)
				l.logError(errors.Wrap(err, text), lvl, depth)
				return
			}
		}
		if lvl == levelError || lvl == levelCritical {
			err := errors.Errorf(text, params...)
			l.logError(ErrWithStacktrace(err, depth), lvl, depth)
		} else {
			l.logWithFallback(lvl, depth, "message", fmt.Sprintf(text, params...))
		}
	} else {
		l.logError(errors.New("first parameter to logger needs to be a string or an error"), levelCritical, depth)
	}
}

// Error logs an error with its stacktrace.
//
// Usage:
//
// Error()
// Error("Normal")
// Error("Params: %d", 100)
// Error(err)
// Error(err, "There was some error")
// Error(err, "There was some error with params: %d", 100)
func (l logger) Error(params ...interface{}) {
	l.logFancy(0, levelError, params...)
}

// Warn prints a warning message
func (l logger) Warn(params ...interface{}) {
	l.logFancy(0, levelWarn, params...)
}

// Info prints an Info message
func (l logger) Info(params ...interface{}) {
	l.logFancy(0, levelInfo, params...)
}

// Notice prints a Notice message
func (l logger) Notice(params ...interface{}) {
	l.logFancy(0, levelNotice, params...)
}

// Debug prints a Debug message
func (l logger) Debug(params ...interface{}) {
	l.logFancy(0, levelDebug, params...)
}

// Fatal logs an error with its stacktrace, then terminates the program.
func (l logger) Fatal(params ...interface{}) {
	l.logFancy(0, levelCritical, params...)
	os.Exit(1)
}

// With returns a logger with the key-value pair added to it
func (l logger) With(key string, value interface{}) Logger {
	l.gklogger = log.With(l.gklogger, key, value)
	return &l
}

func (l logger) WithPrefix(prefix string) Logger {
	l.prefix = prefix
	return &l
}

// WithErrorCounter increments a (prometheus) metric for every error logged.
func (l logger) WithErrorCounter(metric incrementable) Logger {
	l.errorCounter = metric
	return &l
}

func (l logger) incMetric() {
	if l.errorCounter != nil {
		l.errorCounter.Inc()
	}
}

// GetGoKitLogger returns the underlying go-kit-logger
func (l logger) GetGoKitLogger() log.Logger {
	gklogger := levelfilter.Filter(l.loglevel.ToGokitFormat(), l.gklogger)
	return log.With(gklogger, "caller", log.Caller(3))
}

// Log satisfies go-kit logger interface for backwards compatibility.
func (l logger) Log(keyvals ...interface{}) error {
	lvl := levelDebug

	var remaining []interface{}

	for i, k := range keyvals {

		// Every second entry is a value; skip those
		if i%2 != 0 {
			continue
		}

		if len(keyvals) <= i+1 {
			// This key doesn't have a value; That's pretty bad ;(
			continue
		}

		if key, ok := k.(string); ok {
			key := strings.ToLower(key)
			if key == "level" || key == "severity" {
				str, ok := keyvals[i+1].(string)
				if !ok {
					if v, ok := keyvals[i+1].(gklevel.Value); ok {
						str = v.String()
					} else {
						// ok, something clearly isn't right, let's just skip this
						continue
					}
				}

				lvl, _ = loglevelFromString(str)

				if !l.isAllowed(lvl) {
					return nil
				}

				// don't add the level thing into our remaining array
				continue
			} else {
				if key == "msg" {
					key = "message"
				}

				if key == "ts" {
					key = "time"
				}

				remaining = append(remaining, key, keyvals[i+1])
				continue
			}
		}

		remaining = append(remaining, k, keyvals[i+1])
	}

	l.logWithFallback(lvl, 1, remaining...)
	return nil
}

// Legacy wraps a given gokit logger with this logger. If the given logger is this logger already, it will just use it
// directly.
func Legacy(gklogger log.Logger) Logger {
	if l, ok := gklogger.(Logger); ok {
		return l
	}

	return &logger{
		gklogger: gklogger,
		// let all levels pass through so the underlying loggers can make this decision based on their loglevels
		loglevel:        allLevelsFrom(levelDebug),
		useLegacyLevels: true,
	}
}

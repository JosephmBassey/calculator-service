package logger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

// stackTracer is required in order to determine if an error already has a stack
type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

type stacktrace struct {
	st stackTracer
}

func (s stacktrace) getLines() []string {
	var lines []string
	for _, frame := range s.st.StackTrace() {
		pc := uintptr(frame) - 1
		fn := runtime.FuncForPC(pc)
		file, ln := fn.FileLine(pc)
		lines = append(lines, fmt.Sprintf("%s @ %s:%d", fn.Name(), file, ln))
	}
	return lines
}

// getLinesOtherFmt returns the lines in a slightly different format
func (s stacktrace) getTextFormat(useColors bool) string {
	var lines []string

	c := color.New(color.FgHiRed, color.Underline)

	for _, frame := range s.st.StackTrace() {
		pc := uintptr(frame) - 1
		fn := runtime.FuncForPC(pc)
		file, ln := fn.FileLine(pc)
		if useColors {
			lines = append(lines, color.HiRedString(fmt.Sprintf("\t%s", fn.Name())), "\t\t"+c.Sprintf("%s:%d", file, ln))
		} else {
			lines = append(lines, fmt.Sprintf("\t%s", fn.Name()), "\t\t"+fmt.Sprintf("%s:%d", file, ln))
		}

	}
	return strings.Join(lines, "\n") + "\n"
}

// MarshalText puts this stacktrace into a List-Like format for the LogfmtLogger
func (s stacktrace) MarshalText() (text []byte, err error) {
	lines := s.getLines()
	return []byte(fmt.Sprintf("[%s]", strings.Join(lines, " <- "))), nil
}

// MarshalJSON puts this stacktrace into an array for the JsonLogger
func (s stacktrace) MarshalJSON() (text []byte, err error) {
	lines := s.getLines()
	return []byte(fmt.Sprintf(`["%s"]`, strings.Join(lines, `","`))), nil
}

type errWithStacktrace struct {
	cause error
	stack []uintptr
}

func (e *errWithStacktrace) Cause() error {
	return e.cause
}

func (e *errWithStacktrace) StackTrace() errors.StackTrace {
	f := make([]errors.Frame, len(e.stack))
	for i := 0; i < len(f); i++ {
		f[i] = errors.Frame((e.stack)[i])
	}
	return f
}

func (e *errWithStacktrace) Error() string {
	return e.cause.Error()
}

func ErrWithStacktrace(err error, depth int) *errWithStacktrace {
	return &errWithStacktrace{
		cause: err,
		stack: callers(depth + 5),
	}
}

func callers(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[0:n]
}

type causer interface {
	error
	Cause() error
}

func getUnderlyingStacktrace(err error) stackTracer {
	var stErr stackTracer

	for {
		if st, ok := err.(stackTracer); ok {
			stErr = st
		}
		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else {
			break
		}
	}

	return stErr
}

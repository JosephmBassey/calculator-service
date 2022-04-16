package logger

import (
	"fmt"
	"regexp"

	"bytes"
	"strings"

	"io"

	"github.com/fatih/color"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

type customFormatLogger struct {
	fmtstring string
	writer    io.Writer
	pat       *regexp.Regexp
	useColors bool
}

func (l *customFormatLogger) Log(keyvals ...interface{}) error {
	if l.pat == nil {
		l.pat = regexp.MustCompile(`\$[A-Za-z_]+`)
	}
	var key string

	kv := make(map[string]string)

	var params []string

	var lvl level
	var st *stacktrace

	for i, v := range keyvals {
		if i%2 == 0 {
			key = fmt.Sprint(v)
		} else {
			if key != "time" &&
				key != "source_location" &&
				key != "name" &&
				key != "commit" &&
				key != "stacktrace" &&
				key != "severity" &&
				key != "caller" &&
				key != "message" {
				params = append(params, "$"+strings.ToLower(key))
			}

			if key == "severity" {
				lvl, _ = loglevelFromString(fmt.Sprint(v))
			} else if key == "stacktrace" {
				if s, ok := v.(stacktrace); ok {
					st = &s
				}
			}

			kv["$"+strings.ToLower(key)] = fmt.Sprint(v)
		}
	}

	kv["$severity_s"] = severityShort(kv["$severity"])

	exclude := make(map[string]struct{})

	out := string(l.pat.ReplaceAllFunc([]byte(l.fmtstring), func(bytes []byte) []byte {
		k := strings.ToLower(string(bytes))
		exclude[k] = struct{}{}
		str := kv[k]
		if k == "$message" && l.useColors {
			str = useColorForLevel(lvl, str)
		}
		return []byte(str)
	}))

	p := logparams(params, kv, exclude)

	if p != "" {
		p = "(" + p + ")"
	}

	out = strings.Replace(out, "$.", p, -1) + "\n"

	if st != nil {
		if l.useColors {
			out += color.HiRedString("Stack Trace:\n") + st.getTextFormat(true)
		} else {
			out += "Stack Trace:\n" + st.getTextFormat(false)
		}
	}

	_, err := l.writer.Write([]byte(out))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func severityShort(sev string) string {
	switch sev {
	case "debug":
		return "D"
	case "info":
		return "I"
	case "notice":
		return "N"
	case "warn":
		return "W"
	case "error":
		return "E"
	case "critical":
		return "!"
	}

	return "?"
}

func logparams(params []string, kv map[string]string, exclude map[string]struct{}) string {
	bb := bytes.Buffer{}
	logfmtlogger := log.NewLogfmtLogger(&bb)
	var p []interface{}

	for _, param := range params {
		v := kv[param]

		if _, ok := exclude[param]; !ok {
			p = append(p, param, v)
		}
	}

	logfmtlogger.Log(p...)

	return strings.TrimSuffix(bb.String(), "\n")
}

func useColorForLevel(lvl level, str string) string {
	switch lvl {
	case levelDebug:
		return color.CyanString(str)
	case levelError, levelCritical:
		c := color.New(color.Bold, color.Underline, color.FgRed)
		return c.Sprint(str)
	case levelWarn:
		c := color.New(color.Bold, color.Underline, color.FgYellow)
		return c.Sprint(str)
	case levelInfo:
		return color.GreenString(str)
	case levelNotice:
		c := color.New(color.Bold, color.Underline, color.FgGreen)
		return c.Sprint(str)
	}
	return str
}

package httputils

import (
	"context"
	stdlog "log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/log"
)

var (
	defaultReadTimeout  time.Duration = 5 * time.Second
	defaultWriteTimeout time.Duration = 10 * time.Second
	defaultIdleTimeout  time.Duration = 120 * time.Second
)

// WebCtxValue represents the type of value for the context key.
type WebCtxKey int

// KeyValues is how request values are stored/retrieved.
const WebCtxValue WebCtxKey = 1

// WebCtxValues represent state for each request.
type WebCtxValues struct {
	TraceID    string
	Gateway    string
	Now        time.Time
	StatusCode int
}

// SetWebContext set the state for each request.
func SetWebContext(parent context.Context, key WebCtxKey, value WebCtxValues) context.Context {
	ctx := context.WithValue(parent, key, &value)
	return ctx
}

// GetRemoteIP will return the remote IP of the http.Request. It will look for valid
// X-Forwarded-For headers.
// https://cloud.google.com/compute/docs/load-balancing/http/#components
// "X-Forwarded-For: <unverified IP(s)>, <immediate client IP>, <global forwarding rule external IP>, <proxies running in GCP> (requests only)"
// "A comma-separated list of IP addresses appended by the intermediaries the request traveled through. If you are running proxies inside GCP that append data to the X-Forwarded-For header, then your software must take into account the existence and number of those proxies. Only the <immediate client IP> and <global forwarding rule external IP> entries are provided by the load balancer. All other entries in the list are passed along without verification. The <immediate client IP> entry is the
// client that connected directly to the load balancer."
func GetRemoteIP(r *http.Request) net.IP {
	return GetRemoteIPWithOffset(r, 1)
}

// GetRemoteIPWithOffset will return the remote IP of the http.Request, skipping
// the last N X-Forwarded-For entries.
// see https://github.com/didip/tollbooth/blob/master/libstring/libstring.go#L21
func GetRemoteIPWithOffset(r *http.Request, xffOffset int) net.IP {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i, p := range ips {
			ips[i] = strings.TrimSpace(p)
		}
		partIndex := len(ips) - 1 - xffOffset
		if partIndex < 0 {
			partIndex = 0
		}
		ip := net.ParseIP(ips[partIndex])
		if ip != nil {
			return ip
		}
	}

	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip := net.ParseIP(host)
			if ip != nil {
				return ip
			}
		}
	}

	// if all else fails use default from APIPA (Automatic Private IP Addressing) range
	return net.IPv4(169, 254, 0, 1)
}

// GetServerName will look up an Host header in the http.Request
func GetServerName(r *http.Request) string {
	sn := r.Header.Get("Host")
	if sn != "" {
		return sn
	}

	// if all else fails use localhost as a reasonably safe default
	return "localhost"
}

func NewServerWithDefaultTimeouts(logger log.Logger) *http.Server {
	return newServerWithTimeouts(
		logger,
		defaultReadTimeout,
		defaultWriteTimeout,
		defaultIdleTimeout,
	)
}

func NewServerWithTimeouts(logger log.Logger, readTimeout, writeTimeout, idleTimeout time.Duration) *http.Server {
	return newServerWithTimeouts(logger, readTimeout, writeTimeout, idleTimeout)
}

func newServerWithTimeouts(logger log.Logger, readTimeout, writeTimeout, idleTimeout time.Duration) *http.Server {
	l := stdlog.New(log.NewStdlibAdapter(logger), "", stdlog.LstdFlags)
	return &http.Server{
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		ErrorLog:     l,
	}
}

func NewServerWithoutTimeouts(logger log.Logger) *http.Server {
	l := stdlog.New(log.NewStdlibAdapter(logger), "", stdlog.LstdFlags)
	return &http.Server{
		ErrorLog: l,
	}
}

func AddDefaultTimeouts(s *http.Server, logger log.Logger) {
	addTimeouts(s, logger, defaultReadTimeout, defaultWriteTimeout, defaultIdleTimeout)
}

func AddTimeouts(s *http.Server, logger log.Logger, readTimeout, writeTimeout, idleTimeout time.Duration) {
	addTimeouts(s, logger, readTimeout, writeTimeout, idleTimeout)
}

func addTimeouts(s *http.Server, logger log.Logger, readTimeout, writeTimout, idleTimout time.Duration) {
	if s.ReadTimeout == 0 {
		s.ReadTimeout = readTimeout
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = writeTimout
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = idleTimout
	}
	if s.ErrorLog == nil && logger != nil {
		l := stdlog.New(log.NewStdlibAdapter(logger), "", stdlog.LstdFlags)
		s.ErrorLog = l
	}
}

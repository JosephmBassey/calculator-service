// Package depstatus implements a dependency registry
package depstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	DefaultHandler *StatusHandler
)

func init() {
	DefaultHandler = New()
}

// StatusResponse is part of the json response
type StatusResponse struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error"`
}

// StatusResponses is the json response
type StatusResponses struct {
	Deps       []StatusResponse `json:"deps"`
	AllHealthy bool             `json:"all_healthy"`
}

// StatusProvider is the interface that our dependencies have to implement
// to get their status displayed by this status handler
type StatusProvider interface {
	fmt.Stringer
	Status(ctx context.Context) error
}

// StatusHandler is a status handler that will check all registered dependencies
// on each request
type StatusHandler struct {
	deps []StatusProvider
}

// New creates a new status handler
func New() *StatusHandler {
	sh := &StatusHandler{
		deps: make([]StatusProvider, 0),
	}
	gf := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "depstatus_unhealthy_deps_total",
		Help: "Number of unhealthy deps",
	}, func() float64 {
		s := sh.checkDeps(context.Background())
		num := 0
		for _, dep := range s.Deps {
			if !dep.Healthy {
				num++
			}
		}
		return float64(num)
	})
	prometheus.Register(gf)
	return sh
}

// Register will register a new status provider
func (h *StatusHandler) Register(dep StatusProvider) {
	h.deps = append(h.deps, dep)
}

// Register will register a new status provider
func Register(dep StatusProvider) {
	DefaultHandler.Register(dep)
}

func (h *StatusHandler) checkDeps(ctx context.Context) *StatusResponses {
	s := &StatusResponses{
		Deps:       make([]StatusResponse, 0, len(h.deps)),
		AllHealthy: true,
	}
	for _, dep := range h.deps {
		status := StatusResponse{
			Name:    dep.String(),
			Healthy: true,
			Error:   "",
		}
		if err := dep.Status(ctx); err != nil {
			status.Healthy = false
			status.Error = err.Error()
			s.AllHealthy = false
		}
		s.Deps = append(s.Deps, status)
	}
	return s
}

// ServeHTTP implements http.Handler
func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := h.checkDeps(r.Context())
	json.NewEncoder(w).Encode(s)
}

// Handler returns the default handler
func Handler() http.Handler {
	return DefaultHandler
}

package prom

import "github.com/prometheus/client_golang/prometheus"

// MustRegisterOrGet provides an implementation of the deprecated methods
// that was once available from the Prometheus client_golang directly
func MustRegisterOrGet(c prometheus.Collector) prometheus.Collector {
	err := prometheus.Register(c)
	if err == nil {
		return c
	}
	if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
		return are.ExistingCollector
	}
	panic(err)
}

// textOrDefault returns the help text, or if it's not defined it returns a default help text based on the
// second parameter.
func textOrDefault(help, name string) string {
	if help == "" {
		return "No help for " + name
	}
	return help
}

// NewCounter is a wrapper around prometheus.NewCounter that also registers the metric.
func NewCounter(name, help string) prometheus.Counter {
	metric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: textOrDefault(help, name),
	})
	return MustRegisterOrGet(metric).(prometheus.Counter)
}

// NewCounterVec is a wrapper around prometheus.NewCounterVec that also registers the metric.
func NewCounterVec(name, help string, params ...string) *prometheus.CounterVec {
	metric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: textOrDefault(help, name),
	}, params)
	return MustRegisterOrGet(metric).(*prometheus.CounterVec)
}

// NewGauge is a wrapper around prometheus.NewGauge that also registers the metric.
func NewGauge(name, help string) prometheus.Gauge {
	metric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: textOrDefault(help, name),
	})
	return MustRegisterOrGet(metric).(prometheus.Gauge)
}

// NewGaugeVec is a wrapper around prometheus.NewGaugeVec that also registers the metric.
func NewGaugeVec(name, help string, params ...string) *prometheus.GaugeVec {
	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: textOrDefault(help, name),
	}, params)
	return MustRegisterOrGet(metric).(*prometheus.GaugeVec)
}

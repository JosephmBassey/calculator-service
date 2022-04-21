package prom

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InstrumentRoundTripper instruments an http.RoundTripper with a default
// set of metrics
func InstrumentRoundTripper(name string, next http.RoundTripper) http.RoundTripper {
	if name == "" {
		name = "default"
	}
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "http_client_in_flight_requests",
		Help:        "A gauge of in-flight requests for the wrapped client.",
		ConstLabels: map[string]string{"name": name},
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_client_requests_total",
			Help:        "A counter for requests from the wrapped client.",
			ConstLabels: map[string]string{"name": name},
		},
		[]string{"code", "method"},
	)

	// dnsLatencyVec uses custom buckets based on expected dns durations.
	// It has an instance label "event", which is set in the
	// DNSStart and DNSDonehook functions defined in the
	// InstrumentTrace struct below.
	dnsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_client_trace_dns_duration_seconds",
			Help:        "Trace dns latency histogram.",
			ConstLabels: map[string]string{"name": name},
			Buckets:     []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)

	// tlsLatencyVec uses custom buckets based on expected tls durations.
	// It has an instance label "event", which is set in the
	// TLSHandshakeStart and TLSHandshakeDone hook functions defined in the
	// InstrumentTrace struct below.
	tlsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_client_trace_tls_duration_seconds",
			Help:        "Trace tls latency histogram.",
			ConstLabels: map[string]string{"name": name},
			Buckets:     []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
	)

	// histVec has no labels, making it a zero-dimensional ObserverVec.
	histVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_client_request_duration_seconds",
			Help:        "A histogram of request latencies.",
			ConstLabels: map[string]string{"name": name},
			Buckets:     prometheus.DefBuckets,
		},
		[]string{},
	)

	// Register all of the metrics in the standard registry.
	counter = MustRegisterOrGet(counter).(*prometheus.CounterVec)
	tlsLatencyVec = MustRegisterOrGet(tlsLatencyVec).(*prometheus.HistogramVec)
	dnsLatencyVec = MustRegisterOrGet(dnsLatencyVec).(*prometheus.HistogramVec)
	histVec = MustRegisterOrGet(histVec).(*prometheus.HistogramVec)
	inFlightGauge = MustRegisterOrGet(inFlightGauge).(prometheus.Gauge)
	// Define functions for the available httptrace.ClientTrace hook
	// functions that we want to instrument.
	trace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_start")
		},
		DNSDone: func(t float64) {
			dnsLatencyVec.WithLabelValues("dns_done")
		},
		TLSHandshakeStart: func(t float64) {
			tlsLatencyVec.WithLabelValues("tls_handshake_start")
		},
		TLSHandshakeDone: func(t float64) {
			tlsLatencyVec.WithLabelValues("tls_handshake_done")
		},
	}

	// Wrap the default RoundTripper with middleware.
	return promhttp.InstrumentRoundTripperInFlight(inFlightGauge,
		promhttp.InstrumentRoundTripperCounter(counter,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(histVec, next),
			),
		),
	)
}

// PLEASE DO NOT CHANGE THESE BUCKETS WITHOUT CONSENT!
var (
	// DefaultDurations provides a default set of buckets for measuring latencies
	DefaultDurations = []float64{0.001, 0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 1.0, 2, 5}
	// DefaultRequestByteBuckets provides a default set of buckets for measuring request sizes
	DefaultRequestByteBuckets = prometheus.ExponentialBuckets(50, 4.0, 10)
	// DefaultResponseByteBuckets provides a default set of buckets for measuring response sizes
	DefaultResponseByteBuckets = prometheus.ExponentialBuckets(50, 4.0, 10)
)

// InstrumentHandler instruments an http.Handler with a default
// set of metrics. These WILL NOT be suitable for every kind of
// service. If those metrics (e.g. the histogram buckets) don't suit
// your application you are encouraged to create your own set of metrics
// within your service.
func InstrumentHandler(name string, next http.Handler) http.Handler {
	return InstrumentHandlerWithBuckets(
		name,
		next,
		DefaultDurations,
		DefaultRequestByteBuckets,
		DefaultResponseByteBuckets,
	)
}

// InstrumentHandlerWithBuckets instruments an http.Handler with a default
// set of metrics and custom buckets. This allows the histogram buckets to
// be easily adjusted to the specific use case.
func InstrumentHandlerWithBuckets(name string, next http.Handler, durB, reqB, respB []float64) http.Handler {
	constLabelFunc := func() prometheus.Labels {
		return prometheus.Labels{"handler": name}
	}
	return InstrumentHandlerWithBucketsAndLabels(constLabelFunc, next, durB, reqB, respB)
}

// InstrumentHandlerWithBucketsAndLabels instruments an http.Handler with a default
// set of metrics and custom buckets. This allows the histogram buckets to
// be easily adjusted to the specific use case. The const label pairs can be specified
// via the constLabelFunc
func InstrumentHandlerWithBucketsAndLabels(constLabelFunc func() prometheus.Labels, next http.Handler, durB, reqB, respB []float64) http.Handler {
	inFlightGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "http_in_flight_requests",
		Help:        "A gauge of requests currently being served by the wrapped handler.",
		ConstLabels: constLabelFunc(),
	})

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "A counter for requests to the wrapped handler.",
			ConstLabels: constLabelFunc(),
		},
		[]string{"code", "method"},
	)

	durationVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "A histogram of latencies for requests.",
			Buckets:     durB,
			ConstLabels: constLabelFunc(),
		},
		[]string{"code", "method"},
	)

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_response_size_bytes",
			Help:        "A histogram of response sizes for requests.",
			Buckets:     respB,
			ConstLabels: constLabelFunc(),
		},
		[]string{"code", "method"},
	)

	requestSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_size_bytes",
			Help:        "A histogram of request sizes for requests.",
			Buckets:     reqB,
			ConstLabels: constLabelFunc(),
		},
		[]string{"code", "method"},
	)

	// Register all of the metrics in the standard registry.
	inFlightGauge = MustRegisterOrGet(inFlightGauge).(prometheus.Gauge)
	counter = MustRegisterOrGet(counter).(*prometheus.CounterVec)
	durationVec = MustRegisterOrGet(durationVec).(*prometheus.HistogramVec)
	responseSize = MustRegisterOrGet(responseSize).(*prometheus.HistogramVec)
	requestSize = MustRegisterOrGet(requestSize).(*prometheus.HistogramVec)

	// Wrap the pushHandler with our shared middleware, but use the
	// endpoint-specific pushVec with InstrumentHandlerDuration.
	return promhttp.InstrumentHandlerInFlight(inFlightGauge,
		promhttp.InstrumentHandlerCounter(counter,
			promhttp.InstrumentHandlerDuration(durationVec,
				promhttp.InstrumentHandlerResponseSize(responseSize,
					promhttp.InstrumentHandlerRequestSize(requestSize,
						InstrumentedHandlerSchemaCounter(next),
					),
				),
			),
		),
	)
}

// InstrumentedHandlerSchemaCounter adds a count per request schema instrumentation
func InstrumentedHandlerSchemaCounter(next http.Handler) http.HandlerFunc {
	schemaCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_count",
		Help: "Request counter partitioned by schema",
	}, []string{"schema"})
	schemaCounter = MustRegisterOrGet(schemaCounter).(*prometheus.CounterVec)
	return InstrumentedHandlerRequestLabelCounter(
		schemaCounter,
		func(r *http.Request) prometheus.Labels {
			schema := "http"
			if r.TLS != nil {
				schema = "https"
			}
			return prometheus.Labels{"schema": schema}
		},
		next,
	)
}

// InstrumentedHandlerRequestLabelCounter adds a count per label instrumentation. The labels
// are retrieved from the http request via the provided labelRetriever func
func InstrumentedHandlerRequestLabelCounter(c *prometheus.CounterVec, labelRetriever func(*http.Request) prometheus.Labels, next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		labels := labelRetriever(r)
		c.With(labels).Inc()
	})
}

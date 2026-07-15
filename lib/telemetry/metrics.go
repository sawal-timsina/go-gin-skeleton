package telemetry

import (
	"strconv"
	"time"

	"boilerplate-api/lib/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the RED (Rate, Errors, Duration) instruments plus a private
// registry. A dedicated registry (rather than the global default) keeps the
// exposed series deterministic and avoids clashes when tests build the app
// more than once in a process.
type Metrics struct {
	registry    *prometheus.Registry
	reqTotal    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
	inflight    prometheus.Gauge
	Enabled     bool
}

// NewMetrics constructs the metrics collectors labelled with the service name.
func NewMetrics(env config.Env) Metrics {
	reg := prometheus.NewRegistry()
	constLabels := prometheus.Labels{"service": env.ServiceName}

	reqTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total HTTP requests by method, route and status.",
			ConstLabels: constLabels,
		},
		[]string{"method", "route", "status"},
	)
	reqDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request latency in seconds by method and route.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method", "route"},
	)
	inflight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "http_requests_in_flight",
		Help:        "Number of HTTP requests currently being served.",
		ConstLabels: constLabels,
	})

	reg.MustRegister(
		reqTotal,
		reqDuration,
		inflight,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return Metrics{
		registry:    reg,
		reqTotal:    reqTotal,
		reqDuration: reqDuration,
		inflight:    inflight,
		Enabled:     env.MetricsEnabled,
	}
}

// Middleware records rate, errors and duration for every request. It uses the
// matched route template (c.FullPath()) rather than the raw path so metrics
// stay low-cardinality — /users/:id collapses to one series, not one per id.
func (m Metrics) Middleware() gin.HandlerFunc {
	if !m.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		start := time.Now()
		m.inflight.Inc()

		c.Next()

		m.inflight.Dec()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.reqTotal.WithLabelValues(c.Request.Method, route, status).Inc()
		m.reqDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// Handler serves the Prometheus exposition endpoint for /metrics.
func (m Metrics) Handler() gin.HandlerFunc {
	h := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pactus_tracker",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pactus_tracker",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Node check metrics
	NodeCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pactus_tracker",
			Subsystem: "node",
			Name:      "check_total",
			Help:      "Total number of node checks",
		},
		[]string{"node_type", "status"},
	)

	NodeCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pactus_tracker",
			Subsystem: "node",
			Name:      "check_duration_seconds",
			Help:      "Node check duration in seconds",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"node_type"},
	)

	NodeHealthScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pactus_tracker",
			Subsystem: "node",
			Name:      "health_score",
			Help:      "Current health score of nodes",
		},
		[]string{"node_type", "node_name"},
	)

	ActiveNodesCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pactus_tracker",
			Subsystem: "node",
			Name:      "active_count",
			Help:      "Number of active nodes",
		},
		[]string{"node_type"},
	)

	// Database metrics
	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pactus_tracker",
			Subsystem: "database",
			Name:      "connections_active",
			Help:      "Number of active database connections",
		},
	)

	DatabaseConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pactus_tracker",
			Subsystem: "database",
			Name:      "connections_idle",
			Help:      "Number of idle database connections",
		},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pactus_tracker",
			Subsystem: "database",
			Name:      "query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)

	DatabaseErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pactus_tracker",
			Subsystem: "database",
			Name:      "errors_total",
			Help:      "Total number of database errors",
		},
		[]string{"error_type"},
	)

	// Scheduler metrics
	SchedulerJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pactus_tracker",
			Subsystem: "scheduler",
			Name:      "jobs_total",
			Help:      "Total number of scheduled jobs executed",
		},
		[]string{"job_name", "status"},
	)

	SchedulerJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pactus_tracker",
			Subsystem: "scheduler",
			Name:      "job_duration_seconds",
			Help:      "Scheduled job execution duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"job_name"},
	)

	LastSchedulerJobTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pactus_tracker",
			Subsystem: "scheduler",
			Name:      "last_job_timestamp",
			Help:      "Unix timestamp of last job execution",
		},
		[]string{"job_name"},
	)

	// Rate limiter metrics
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pactus_tracker",
			Subsystem: "rate_limiter",
			Name:      "requests_total",
			Help:      "Total number of rate-limited requests",
		},
		[]string{"ip", "allowed"},
	)
)

// Metrics provides convenience methods for recording metrics
type Metrics struct{}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordHTTPRequest records an HTTP request metric
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	HttpRequestsTotal.WithLabelValues(method, endpoint, http.StatusText(statusCode)).Inc()
	HttpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordNodeCheck records a node check metric
func (m *Metrics) RecordNodeCheck(nodeType string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "failure"
	}
	NodeCheckTotal.WithLabelValues(nodeType, status).Inc()
	NodeCheckDuration.WithLabelValues(nodeType).Observe(duration.Seconds())
}

// UpdateNodeHealthScore updates a node's health score
func (m *Metrics) UpdateNodeHealthScore(nodeType, nodeName string, score float64) {
	NodeHealthScore.WithLabelValues(nodeType, nodeName).Set(score)
}

// UpdateActiveNodesCount updates the count of active nodes
func (m *Metrics) UpdateActiveNodesCount(nodeType string, count int) {
	ActiveNodesCount.WithLabelValues(nodeType).Set(float64(count))
}

// RecordDatabaseQuery records a database query metric
func (m *Metrics) RecordDatabaseQuery(queryType string, duration time.Duration) {
	DatabaseQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

// RecordDatabaseError records a database error
func (m *Metrics) RecordDatabaseError(errorType string) {
	DatabaseErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordSchedulerJob records a scheduler job execution
func (m *Metrics) RecordSchedulerJob(jobName string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "failure"
	}
	SchedulerJobsTotal.WithLabelValues(jobName, status).Inc()
	SchedulerJobDuration.WithLabelValues(jobName).Observe(duration.Seconds())
	LastSchedulerJobTime.WithLabelValues(jobName).SetToCurrentTime()
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

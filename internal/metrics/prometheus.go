package metrics

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chaowen/budget/internal/repository"
)

type Collector struct {
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	grpcRequestsTotal    *prometheus.CounterVec
	grpcRequestDuration  *prometheus.HistogramVec
	AuthFailuresTotal    *prometheus.CounterVec
	userCount            prometheus.Gauge
	transactionCount     prometheus.Gauge
	healthyNodes         *prometheus.GaugeVec
	metricsUpdateErrors  *prometheus.CounterVec
	metricsLastUpdatedAt prometheus.Gauge
	DBQueriesTotal       *prometheus.CounterVec
	DBQueryDuration      *prometheus.HistogramVec
	DBQueryErrors        *prometheus.CounterVec
	SnapshotsRecorded    *prometheus.CounterVec
}

func NewCollector(reg prometheus.Registerer) *Collector {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	factory := promauto.With(reg)

	return &Collector{
		httpRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_http_requests_total",
				Help: "Total number of HTTP requests handled.",
			},
			[]string{"method", "path", "status"},
		),
		httpRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "budget_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		grpcRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_grpc_requests_total",
				Help: "Total number of gRPC requests handled.",
			},
			[]string{"method", "code"},
		),
		grpcRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "budget_grpc_request_duration_seconds",
				Help:    "gRPC request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "code"},
		),
		userCount: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "budget_user_count",
				Help: "Total number of users.",
			},
		),
		transactionCount: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "budget_transaction_count",
				Help: "Total number of transactions.",
			},
		),
		healthyNodes: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "budget_healthy_nodes",
				Help: "Node health indicator by node (1 = healthy, 0 = unhealthy).",
			},
			[]string{"node"},
		),
		metricsUpdateErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_metrics_update_errors_total",
				Help: "Total number of business metrics update errors.",
			},
			[]string{"metric"},
		),
		AuthFailuresTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_auth_failures_total",
				Help: "Total number of authentication failures.",
			},
			[]string{"reason", "method"},
		),
		metricsLastUpdatedAt: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "budget_metrics_last_updated_unix",
				Help: "Unix timestamp of the latest successful business metrics refresh.",
			},
		),
		DBQueriesTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_db_queries_total",
				Help: "Total number of database queries executed.",
			},
			[]string{"operation", "table"},
		),
		DBQueryDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "budget_db_query_duration_seconds",
				Help:    "Database query duration in seconds.",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"operation", "table"},
		),
		DBQueryErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_db_query_errors_total",
				Help: "Total number of database query errors.",
			},
			[]string{"operation", "table"},
		),
		SnapshotsRecorded: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "budget_asset_snapshots_recorded_total",
				Help: "Total number of asset snapshots recorded.",
			},
			[]string{"trigger"},
		),
	}
}

func (c *Collector) ObserveDBQuery(operation, table string, duration time.Duration, err error) {
	labels := prometheus.Labels{"operation": operation, "table": table}
	c.DBQueriesTotal.With(labels).Inc()
	c.DBQueryDuration.With(labels).Observe(duration.Seconds())
	if err != nil {
		c.DBQueryErrors.With(labels).Inc()
	}
}

func (c *Collector) RecordSnapshot(trigger string) {
	c.SnapshotsRecorded.WithLabelValues(trigger).Inc()
}

func (c *Collector) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		statusCode := strconv.Itoa(recorder.statusCode)
		labels := prometheus.Labels{
			"method": r.Method,
			"path":   normalizePath(r.URL.Path),
			"status": statusCode,
		}

		c.httpRequestsTotal.With(labels).Inc()
		c.httpRequestDuration.With(labels).Observe(time.Since(start).Seconds())
	})
}

func (c *Collector) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		grpcCode := codes.OK
		if err != nil {
			grpcCode = status.Code(err)
		}

		labels := prometheus.Labels{
			"method": info.FullMethod,
			"code":   grpcCode.String(),
		}

		c.grpcRequestsTotal.With(labels).Inc()
		c.grpcRequestDuration.With(labels).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

func (c *Collector) StartBusinessMetricsUpdater(ctx context.Context, db *repository.DB, nodeName string, interval time.Duration) {
	if db == nil || db.Pool == nil {
		return
	}
	if nodeName == "" {
		nodeName = "unknown"
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	update := func() {
		healthy := 0.0
		if err := db.Pool.Ping(ctx); err != nil {
			c.metricsUpdateErrors.WithLabelValues("db_ping").Inc()
		} else {
			healthy = 1
		}
		c.healthyNodes.WithLabelValues(nodeName).Set(healthy)

		if healthy == 0 {
			return
		}

		var users int64
		if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
			c.metricsUpdateErrors.WithLabelValues("user_count").Inc()
		} else {
			c.userCount.Set(float64(users))
		}

		var transactions int64
		if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&transactions); err != nil {
			c.metricsUpdateErrors.WithLabelValues("transaction_count").Inc()
		} else {
			c.transactionCount.Set(float64(transactions))
		}

		c.metricsLastUpdatedAt.Set(float64(time.Now().Unix()))
	}

	update()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				update()
			}
		}
	}()
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}

	parts := strings.Split(path, "/")
	for i := range parts {
		part := parts[i]
		if part == "" {
			continue
		}

		if _, err := uuid.Parse(part); err == nil {
			parts[i] = ":id"
			continue
		}

		if _, err := strconv.ParseInt(part, 10, 64); err == nil {
			parts[i] = ":id"
		}
	}

	return strings.Join(parts, "/")
}

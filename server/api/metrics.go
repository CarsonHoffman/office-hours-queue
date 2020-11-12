package api

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestsCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "requests_count",
		Help: "The number of requests by endpoint and response code.",
	},
	[]string{"method", "path", "code"},
)

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter not Hijacker")
	}
	return h.Hijack()
}

var requestsTimer = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "requests_time",
		Help: "The duration of requests by endpoint and response code.",
	},
	[]string{"method", "path", "code"},
)

var requestsSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "requests_size",
		Help:    "The size of response bodies by endpoint and response code.",
		Buckets: []float64{1, 10, 100, 200, 500, 1000, 2000, 5000, 10000, 100000, 1000000, 10000000, 100000000},
	},
	[]string{"method", "path", "code"},
)

func instrumenter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &StatusRecorder{
			ResponseWriter: w,
			Status:         200,
		}

		start := time.Now()
		next.ServeHTTP(recorder, r)
		duration := time.Since(start)

		requestLabels := prometheus.Labels{
			"method": r.Method,
			"path":   chi.RouteContext(r.Context()).RoutePattern(),
			"code":   strconv.Itoa(recorder.Status),
		}

		requestsCounter.With(requestLabels).Inc()
		requestsTimer.With(requestLabels).Observe(duration.Seconds())

		// If err != nil, assume Content-Length wasn't included, which means
		// we want 0 anyways! Yay zero values!
		responseSize, _ := strconv.Atoi(recorder.Header().Get("Content-Length"))
		requestsSize.With(requestLabels).Observe(float64(responseSize))
	})
}

func init() {
	prometheus.MustRegister(requestsCounter, requestsTimer, requestsSize)
}

func (s *Server) MetricsHandler() http.HandlerFunc {
	handler := promhttp.Handler()
	return func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || pass != s.metricsPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, r)
	}
}

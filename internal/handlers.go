package fleetlock

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// Makes a responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// Writes the header code into the response and makes it available for logging
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

const (
	fleetLockHeaderKey = "fleet-lock-protocol"
	millisecondsInSec  = 1000
)

// POSTHandler returns a handler that requires the POST method.
func POSTHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			encodeReply(w, NewReply(KindMethodNotAllowed, "required method POST"))
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

// HeaderHandler returns a handler that requires a given header key/value.
func HeaderHandler(key, value string, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get(key) != value {
			encodeReply(w, NewReply(KindMissingHeader, "missing required header %s: %s", key, value))
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

// InstrumentedHandler returns a handler that instruments http requests
func InstrumentedHandler(totalRequests, responseStatus *prometheus.CounterVec, httpDuration *prometheus.HistogramVec, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			ms := v * millisecondsInSec // make microseconds
			httpDuration.WithLabelValues(path).Observe(ms)
		}))
		defer timer.ObserveDuration()
		rw := newResponseWriter(w)
		next.ServeHTTP(rw, req)

		responseStatus.With(prometheus.Labels{
			"path":   path,
			"status": strconv.Itoa(rw.statusCode),
		}).Inc()
		totalRequests.WithLabelValues(path).Inc()
	}
	return http.HandlerFunc(fn)
}

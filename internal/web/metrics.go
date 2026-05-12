package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Métriques HTTP globales exposées sur /metrics.
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "discordbot",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Nombre total de requêtes HTTP par méthode, route et code de statut.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "discordbot",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Durée des requêtes HTTP en secondes.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
)

// metricsMiddleware instrumente chaque requête HTTP pour Prometheus.
// Il utilise le pattern chi pour obtenir la route normalisée (ex: /guilds/{guildID}).
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(wrapped, r)

		ctx := chi.RouteContext(r.Context())
		route := "/unknown"
		if ctx != nil && ctx.RoutePattern() != "" {
			route = ctx.RoutePattern()
		}

		status := strconv.Itoa(wrapped.Status())
		elapsed := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(elapsed)
	})
}

// metricsHandler retourne le handler Prometheus standard.
func metricsHandler() http.Handler {
	return promhttp.Handler()
}

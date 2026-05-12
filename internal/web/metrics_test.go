package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsEndpoint(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/metrics", metricsHandler().ServeHTTP)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Le Content-Type doit être text/plain (format Prometheus exposition).
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")
}

func TestMetricsMiddlewareInstruments(t *testing.T) {
	// On réinitialise les compteurs en créant un registre isolé pour le test.
	// On vérifie simplement que le middleware ne panique pas et laisse passer la requête.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := chi.NewRouter()
	r.Use(metricsMiddleware)
	r.Get("/healthz", next.ServeHTTP)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

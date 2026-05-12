package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := newRateLimiter(10, 3) // 10 req/s, burst=3

	// Les 3 premières requêtes passent (burst).
	for i := 0; i < 3; i++ {
		assert.True(t, rl.allow("1.2.3.4"), "requête %d devrait passer", i+1)
	}
	// La 4ème est bloquée.
	assert.False(t, rl.allow("1.2.3.4"), "la 4ème requête devrait être bloquée")

	// Une autre IP n'est pas affectée.
	assert.True(t, rl.allow("5.6.7.8"), "IP différente devrait passer")
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := newRateLimiter(10, 5)
	rl.allow("1.2.3.4")
	assert.Equal(t, 1, len(rl.buckets))

	// Cleanup immédiat avec TTL=0 : tout doit être supprimé.
	rl.cleanup(0)
	assert.Equal(t, 0, len(rl.buckets))
}

func TestRateLimitMiddleware_429(t *testing.T) {
	// burst=1 : la 2ème requête immédiate doit recevoir 429.
	mw := RateLimitMiddleware(1, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := mw(next)

	// 1ère requête : OK.
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Real-IP", "10.0.0.1")
	h.ServeHTTP(rr1, req1)
	require.Equal(t, http.StatusOK, rr1.Code)

	// 2ème requête immédiate : 429.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Real-IP", "10.0.0.1")
	h.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
}

func TestRealIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{"X-Real-IP pris en priorité", map[string]string{"X-Real-IP": "1.2.3.4"}, "9.9.9.9:1234", "1.2.3.4"},
		{"X-Forwarded-For première entrée", map[string]string{"X-Forwarded-For": "5.6.7.8, 9.10.11.12"}, "9.9.9.9:1234", "5.6.7.8"},
		{"RemoteAddr fallback", nil, "127.0.0.1:5000", "127.0.0.1:5000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			assert.Equal(t, tc.expected, realIP(req))
		})
	}
}

package web

import (
	"net/http"
	"sync"
	"time"
)

// rateLimiter est un rate limiter par IP basé sur un token bucket simplifié.
// Chaque IP dispose d'un bucket de `burst` tokens, rechargé de `rate` tokens/seconde.
// Conception volontairement sans dépendance externe (golang.org/x/time/rate est
// une option plus robuste pour un usage production).
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64 // tokens ajoutés par seconde
	burst   float64 // capacité maximale du bucket
}

type tokenBucket struct {
	tokens   float64
	lastSeen time.Time
}

// newRateLimiter crée un rate limiter avec `rate` requêtes/seconde et `burst` en rafale.
func newRateLimiter(rate, burst float64) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
	}
	return rl
}

// allow retourne true si l'IP peut effectuer une requête, false si elle est throttlée.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[ip]
	if !ok {
		rl.buckets[ip] = &tokenBucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	// Recharge les tokens en fonction du temps écoulé.
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// cleanup supprime les buckets inactifs depuis plus de `ttl`.
// À appeler périodiquement pour éviter les fuites mémoire.
func (rl *rateLimiter) cleanup(ttl time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-ttl)
	for ip, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, ip)
		}
	}
}

// RateLimitMiddleware retourne un middleware chi qui limite les requêtes par IP.
// rate : requêtes autorisées par seconde par IP.
// burst : tolérance en rafale.
func RateLimitMiddleware(rate, burst float64) func(http.Handler) http.Handler {
	rl := newRateLimiter(rate, burst)
	// Nettoyage périodique des buckets inactifs (toutes les 5 minutes).
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup(10 * time.Minute)
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !rl.allow(ip) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP extrait l'IP réelle en tenant compte des proxies.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Prendre la première IP de la chaîne.
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' {
				return ip[:i]
			}
		}
		return ip
	}
	return r.RemoteAddr
}

package web

import "net/http"

// securityHeaders ajoute les headers de sécurité HTTP recommandés sur chaque réponse.
// Ces headers protègent contre le clickjacking, le MIME-sniffing et forcent HTTPS.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// Empêche le chargement dans un iframe (protection clickjacking).
		h.Set("X-Frame-Options", "DENY")
		// Empêche le MIME-sniffing par le navigateur.
		h.Set("X-Content-Type-Options", "nosniff")
		// Force HTTPS pour les futures connexions (1 an).
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Désactive le referrer tiers.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy restrictive pour une API JSON pure.
		h.Set("Content-Security-Policy", "default-src 'none'")
		// Désactive les fonctionnalités navigateur non nécessaires.
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

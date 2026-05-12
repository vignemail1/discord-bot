package web

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// requireAuth vérifie qu'une session authentifiée est présente.
// Redirige vers /auth/login si la session est absente ou non authentifiée.
func (srv *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := srv.loadSession(r)
		if sess == nil || !sess.IsAuthenticated() {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		ctx := r.Context()
		ctx = contextWithValue(ctx, contextKeySession, sess)
		ctx = contextWithValue(ctx, contextKeyUserID, sess.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loadSession lit le cookie de session et retourne la session correspondante.
// Retourne nil si le cookie est absent ou si la session est introuvable/expirée.
func (srv *Server) loadSession(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return srv.sessions.Get(cookie.Value)
}

// slogRequest est un middleware chi qui logue chaque requête HTTP en JSON structuré.
func slogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

// contextWithValue est un wrapper générique pour context.WithValue avec les clés typées.
func contextWithValue[V any](ctx interface{ Value(any) any }, key contextKey, val V) interface{ Value(any) any } {
	// On doit retourner un context.Context — utiliser le vrai type.
	return contextSet(ctx, key, val)
}

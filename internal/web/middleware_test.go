package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/config"
)

// newTestServer crée un Server minimal pour les tests middleware.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{
		cfg: &config.Config{
			DiscordClientID:     "test-client-id",
			DiscordClientSecret: "test-client-secret",
			DiscordRedirectURL:  "http://localhost/auth/callback",
		},
		sessions:   NewSessionStore(),
		httpClient: &http.Client{},
	}
}

func TestRequireAuth_NoCookie(t *testing.T) {
	srv := newTestServer(t)

	handler := srv.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/guilds", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	srv := newTestServer(t)

	handler := srv.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/guilds", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid-session-id"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
}

func TestRequireAuth_UnauthenticatedSession(t *testing.T) {
	srv := newTestServer(t)

	// Session créée mais sans UserID (pré-auth).
	sess, err := srv.sessions.Create()
	require.NoError(t, err)

	handler := srv.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/guilds", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestRequireAuth_AuthenticatedSession(t *testing.T) {
	srv := newTestServer(t)

	sess, err := srv.sessions.Create()
	require.NoError(t, err)
	sess.UserID = "123456789"
	sess.Username = "testuser"
	srv.sessions.Save(sess)

	called := false
	handler := srv.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Vérifier que la session est injectée dans le contexte.
		ctxSess := sessionFromContext(r.Context())
		assert.NotNil(t, ctxSess)
		assert.Equal(t, "123456789", ctxSess.UserID)

		userID := userIDFromContext(r.Context())
		assert.Equal(t, "123456789", userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/guilds", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "le handler suivant doit être appelé")
}

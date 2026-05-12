package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/config"
)

func TestHandleLogin_CreatesCookieAndRedirects(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	srv.handleLogin(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)

	loc := rec.Header().Get("Location")
	assert.Contains(t, loc, "discord.com/api/oauth2/authorize")
	assert.Contains(t, loc, "client_id=test-client-id")
	assert.Contains(t, loc, "state=")

	// Le cookie de session doit être positionné.
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	require.NotNil(t, sessionCookie, "cookie de session attendu")
	assert.NotEmpty(t, sessionCookie.Value)

	// La session doit exister dans le store.
	sess := srv.sessions.Get(sessionCookie.Value)
	require.NotNil(t, sess)
	assert.NotEmpty(t, sess.StateToken)
	assert.False(t, sess.IsAuthenticated(), "session non encore authentifiée")
}

func TestHandleCallback_MissingCookie(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=xyz", nil)
	rec := httptest.NewRecorder()
	srv.handleCallback(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
}

func TestHandleCallback_InvalidState(t *testing.T) {
	srv := newTestServer(t)

	sess, err := srv.sessions.Create()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=WRONG_STATE", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	srv.handleCallback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleCallback_DiscordError(t *testing.T) {
	srv := newTestServer(t)

	sess, err := srv.sessions.Create()
	require.NoError(t, err)

	url := "/auth/callback?error=access_denied&error_description=User+denied&state=" + sess.StateToken
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	srv.handleCallback(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
}

func TestHandleLogout_ClearsSessionAndCookie(t *testing.T) {
	srv := newTestServer(t)

	sess, err := srv.sessions.Create()
	require.NoError(t, err)
	sess.UserID = "123"
	srv.sessions.Save(sess)

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	srv.handleLogout(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))

	// La session doit être supprimée du store.
	assert.Nil(t, srv.sessions.Get(sess.ID))

	// Le cookie doit être expiré (MaxAge == -1).
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			assert.Equal(t, -1, c.MaxAge)
		}
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	srv.handleLogout(rec, req)

	// Doit rediriger sans paniquer même sans cookie.
	assert.Equal(t, http.StatusFound, rec.Code)
}

// Vérifie que le cfg est bien passé au newTestServer (utilisé dans oauth2.go).
func TestNewTestServer_Config(t *testing.T) {
	srv := newTestServer(t)
	assert.Equal(t, "test-client-id", srv.cfg.DiscordClientID)
}

// newTestServer redéfini localement dans oauth2_test.go pour accès direct au *config.Config.
func init() {
	_ = &config.Config{} // import utilisé
}

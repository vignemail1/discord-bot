package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	discordAuthURL   = "https://discord.com/api/oauth2/authorize"
	discordTokenURL  = "https://discord.com/api/oauth2/token"
	discordRevokeURL = "https://discord.com/api/oauth2/token/revoke"
	discordAPIBase   = "https://discord.com/api/v10"
	oauth2Scope      = "identify guilds"
)

// DiscordUser est la réponse partielle de GET /users/@me.
type DiscordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

// DiscordGuild est la réponse partielle de GET /users/@me/guilds.
type DiscordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions string `json:"permissions"`
}

// tokenResponse est la réponse de l'endpoint /oauth2/token.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// handleLogin démarre le flux OAuth2 Discord.
// GET /auth/login
func (srv *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	sess, err := srv.sessions.Create()
	if err != nil {
		slog.Error("oauth2: création session échouée", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})

	params := url.Values{}
	params.Set("client_id", srv.cfg.DiscordClientID)
	params.Set("redirect_uri", srv.cfg.DiscordRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", oauth2Scope)
	params.Set("state", sess.StateToken)
	params.Set("prompt", "none")

	http.Redirect(w, r, discordAuthURL+"?"+params.Encode(), http.StatusFound)
}

// handleCallback traite le retour du flux OAuth2 Discord.
// GET /auth/callback?code=...&state=...
func (srv *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Récupérer la session existante via le cookie.
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		slog.Warn("oauth2: callback sans cookie de session")
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	sess := srv.sessions.Get(cookie.Value)
	if sess == nil {
		slog.Warn("oauth2: callback — session introuvable ou expirée", "id", cookie.Value)
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Vérification CSRF.
	state := r.URL.Query().Get("state")
	if state == "" || state != sess.StateToken {
		slog.Warn("oauth2: état CSRF invalide",
			"expected", sess.StateToken, "got", state)
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	// Vérifier qu'il n'y a pas d'erreur OAuth2 renvoyée par Discord.
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.Warn("oauth2: discord a retourné une erreur",
			"error", errParam,
			"description", r.URL.Query().Get("error_description"))
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	// Échange du code contre un access_token.
	tok, err := srv.exchangeCode(r.Context(), code)
	if err != nil {
		slog.Error("oauth2: échange de code échoué", "err", err)
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	// Récupération du profil utilisateur.
	user, err := srv.fetchUser(r.Context(), tok.AccessToken)
	if err != nil {
		slog.Error("oauth2: récupération profil échouée", "err", err)
		http.Error(w, "user fetch failed", http.StatusBadGateway)
		return
	}

	// Mise à jour de la session.
	sess.UserID = user.ID
	sess.Username = user.Username
	sess.GlobalName = user.GlobalName
	sess.AvatarHash = user.Avatar
	sess.AccessToken = tok.AccessToken
	sess.RefreshToken = tok.RefreshToken
	sess.TokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	srv.sessions.Save(sess)

	slog.Info("oauth2: authentification réussie",
		"user_id", user.ID, "username", user.Username)

	http.Redirect(w, r, "/guilds", http.StatusFound)
}

// handleLogout invalide la session et redirige vers /auth/login.
// GET /auth/logout
func (srv *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		sess := srv.sessions.Get(cookie.Value)
		if sess != nil && sess.AccessToken != "" {
			// Révocation best-effort : on ne bloque pas sur l'erreur.
			_ = srv.revokeToken(r.Context(), sess.AccessToken)
		}
		srv.sessions.Delete(cookie.Value)
	}

	// Expire le cookie côté client.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

// exchangeCode échange un code d'autorisation contre un tokenResponse.
func (srv *Server) exchangeCode(ctx context.Context, code string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", srv.cfg.DiscordClientID)
	data.Set("client_secret", srv.cfg.DiscordClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", srv.cfg.DiscordRedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordTokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := srv.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord token endpoint: HTTP %d — %s", resp.StatusCode, body)
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("discord token: décodage JSON: %w", err)
	}
	return &tok, nil
}

// fetchUser appelle GET /users/@me avec l'access_token fourni.
func (srv *Server) fetchUser(ctx context.Context, accessToken string) (*DiscordUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := srv.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord users/@me: HTTP %d — %s", resp.StatusCode, body)
	}

	var u DiscordUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("discord users/@me: décodage JSON: %w", err)
	}
	return &u, nil
}

// fetchGuilds appelle GET /users/@me/guilds avec l'access_token fourni.
func (srv *Server) fetchGuilds(ctx context.Context, accessToken string) ([]DiscordGuild, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPIBase+"/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := srv.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord users/@me/guilds: HTTP %d — %s", resp.StatusCode, body)
	}

	var guilds []DiscordGuild
	if err := json.Unmarshal(body, &guilds); err != nil {
		return nil, fmt.Errorf("discord users/@me/guilds: décodage JSON: %w", err)
	}
	return guilds, nil
}

// revokeToken révoque un token OAuth2 Discord (best-effort).
func (srv *Server) revokeToken(ctx context.Context, accessToken string) error {
	data := url.Values{}
	data.Set("client_id", srv.cfg.DiscordClientID)
	data.Set("client_secret", srv.cfg.DiscordClientSecret)
	data.Set("token", accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordRevokeURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := srv.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}

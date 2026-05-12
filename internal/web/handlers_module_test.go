package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

// newTestServerModules crée un Server minimal avec un ModuleRepositoryMock injecté.
func newTestServerModules(modRepo *mock.ModuleRepositoryMock) *Server {
	return &Server{
		sessions:   NewSessionStore(),
		moduleRepo: modRepo,
	}
}

// routeModuleRequest exécute une requête sur un handler chi avec les paramètres d'URL injectés.
func routeModuleRequest(handler http.HandlerFunc, method, path string, body []byte, params map[string]string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))

	// Injection des paramètres chi dans le contexte.
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler(rr, req)
	return rr
}

// --- handleListModules ---

func TestHandleListModules_OK(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "invitefilter", Enabled: true,
		ConfigJSON: json.RawMessage(`{"key":"val"}`),
	})
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "identityhistory", Enabled: false,
	})

	srv := newTestServerModules(modRepo)
	rr := routeModuleRequest(srv.handleListModules, http.MethodGet, "/guilds/guild1/modules", nil,
		map[string]string{"guildID": "guild1"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []moduleResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestHandleListModules_EmptyGuild(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	srv := newTestServerModules(modRepo)

	rr := routeModuleRequest(srv.handleListModules, http.MethodGet, "/guilds/unknown/modules", nil,
		map[string]string{"guildID": "unknown"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []moduleResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestHandleListModules_RepoError(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	modRepo.ListErr = assert.AnError
	srv := newTestServerModules(modRepo)

	rr := routeModuleRequest(srv.handleListModules, http.MethodGet, "/guilds/guild1/modules", nil,
		map[string]string{"guildID": "guild1"})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- handleSetModuleEnabled ---

func TestHandleSetModuleEnabled_Enable(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "invitefilter", Enabled: false,
	})

	srv := newTestServerModules(modRepo)
	body, _ := json.Marshal(map[string]bool{"enabled": true})
	rr := routeModuleRequest(srv.handleSetModuleEnabled, http.MethodPut,
		"/guilds/guild1/modules/invitefilter", body,
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp moduleResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.True(t, resp.Enabled)
	assert.Equal(t, "invitefilter", resp.ModuleName)
}

func TestHandleSetModuleEnabled_Disable(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "invitefilter", Enabled: true,
	})

	srv := newTestServerModules(modRepo)
	body, _ := json.Marshal(map[string]bool{"enabled": false})
	rr := routeModuleRequest(srv.handleSetModuleEnabled, http.MethodPut,
		"/guilds/guild1/modules/invitefilter", body,
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp moduleResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.False(t, resp.Enabled)
}

func TestHandleSetModuleEnabled_InvalidJSON(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	srv := newTestServerModules(modRepo)

	rr := routeModuleRequest(srv.handleSetModuleEnabled, http.MethodPut,
		"/guilds/guild1/modules/invitefilter", []byte(`not-json`),
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleSetModuleEnabled_SetEnabledError(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	modRepo.SetEnabledErr = assert.AnError
	srv := newTestServerModules(modRepo)

	body, _ := json.Marshal(map[string]bool{"enabled": true})
	rr := routeModuleRequest(srv.handleSetModuleEnabled, http.MethodPut,
		"/guilds/guild1/modules/invitefilter", body,
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- handleUpdateModuleConfig ---

func TestHandleUpdateModuleConfig_OK(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "invitefilter", Enabled: true,
		ConfigJSON: json.RawMessage(`{}`),
	})

	srv := newTestServerModules(modRepo)
	newCfg := json.RawMessage(`{"max_invites":5,"whitelist":["discord.gg"]}`)
	rr := routeModuleRequest(srv.handleUpdateModuleConfig, http.MethodPut,
		"/guilds/guild1/modules/invitefilter/config", newCfg,
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp moduleResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "invitefilter", resp.ModuleName)
	assert.JSONEq(t, `{"max_invites":5,"whitelist":["discord.gg"]}`, string(resp.Config))
}

func TestHandleUpdateModuleConfig_NotAnObject(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	srv := newTestServerModules(modRepo)

	// Tableau JSON — refusé.
	rr := routeModuleRequest(srv.handleUpdateModuleConfig, http.MethodPut,
		"/guilds/guild1/modules/invitefilter/config", []byte(`[1,2,3]`),
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdateModuleConfig_InvalidJSON(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	srv := newTestServerModules(modRepo)

	rr := routeModuleRequest(srv.handleUpdateModuleConfig, http.MethodPut,
		"/guilds/guild1/modules/invitefilter/config", []byte(`{bad json`),
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleUpdateModuleConfig_UpdateConfigError(t *testing.T) {
	modRepo := mock.NewModuleRepository()
	_ = modRepo.Upsert(context.Background(), repository.GuildModule{
		GuildID: "guild1", ModuleName: "invitefilter", Enabled: true,
	})
	modRepo.UpdateCfgErr = assert.AnError
	srv := newTestServerModules(modRepo)

	rr := routeModuleRequest(srv.handleUpdateModuleConfig, http.MethodPut,
		"/guilds/guild1/modules/invitefilter/config", []byte(`{"k":"v"}`),
		map[string]string{"guildID": "guild1", "moduleName": "invitefilter"})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- guildModuleToResponse ---

func TestGuildModuleToResponse_NilConfig(t *testing.T) {
	m := repository.GuildModule{
		GuildID: "g", ModuleName: "mod", Enabled: true,
		ConfigJSON: nil,
	}
	resp := guildModuleToResponse(m)
	assert.Equal(t, json.RawMessage(`{}`), resp.Config)
}

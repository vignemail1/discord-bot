package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// moduleResponse est la vue JSON d'un module pour le dashboard.
type moduleResponse struct {
	ModuleName string          `json:"module_name"`
	Enabled    bool            `json:"enabled"`
	Config     json.RawMessage `json:"config"`
}

func guildModuleToResponse(m repository.GuildModule) moduleResponse {
	cfg := m.ConfigJSON
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return moduleResponse{
		ModuleName: m.ModuleName,
		Enabled:    m.Enabled,
		Config:     cfg,
	}
}

// handleListModules retourne tous les modules enregistrés pour une guilde.
// GET /guilds/{guildID}/modules
func (srv *Server) handleListModules(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")

	mods, err := srv.moduleRepo.ListByGuild(r.Context(), guildID)
	if err != nil {
		slog.Error("handlers: ListByGuild échoué",
			"guild_id", guildID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	result := make([]moduleResponse, 0, len(mods))
	for _, m := range mods {
		result = append(result, guildModuleToResponse(m))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleSetModuleEnabled active ou désactive un module.
// PUT /guilds/{guildID}/modules/{moduleName}
//
// Body JSON attendu : {"enabled": true|false}
func (srv *Server) handleSetModuleEnabled(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")
	moduleName := chi.URLParam(r, "moduleName")

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := srv.moduleRepo.SetEnabled(r.Context(), guildID, moduleName, body.Enabled); err != nil {
		slog.Error("handlers: SetEnabled échoué",
			"guild_id", guildID, "module", moduleName, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mod, err := srv.moduleRepo.Get(r.Context(), guildID, moduleName)
	if err != nil {
		slog.Error("handlers: Get post-SetEnabled échoué",
			"guild_id", guildID, "module", moduleName, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if mod == nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(guildModuleToResponse(*mod))
}

// handleUpdateModuleConfig met à jour la configuration JSON d'un module.
// PUT /guilds/{guildID}/modules/{moduleName}/config
//
// Le body doit être un objet JSON valide (ne remplace que config_json).
func (srv *Server) handleUpdateModuleConfig(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")
	moduleName := chi.URLParam(r, "moduleName")

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	// Garantit que c'est bien un objet JSON, pas un scalaire ou un tableau.
	if len(raw) == 0 || raw[0] != '{' {
		http.Error(w, "config must be a JSON object", http.StatusBadRequest)
		return
	}

	if err := srv.moduleRepo.UpdateConfig(r.Context(), guildID, moduleName, raw); err != nil {
		slog.Error("handlers: UpdateConfig échoué",
			"guild_id", guildID, "module", moduleName, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mod, err := srv.moduleRepo.Get(r.Context(), guildID, moduleName)
	if err != nil {
		slog.Error("handlers: Get post-UpdateConfig échoué",
			"guild_id", guildID, "module", moduleName, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if mod == nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(guildModuleToResponse(*mod))
}

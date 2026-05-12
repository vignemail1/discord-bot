package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// manageGuildPermission est le bit de permission Discord "Gérer le serveur".
const manageGuildPermission = 0x20

// guildResponse est la vue JSON d'une guilde pour le dashboard.
type guildResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Managed     bool   `json:"managed"`    // bot déjà présent dans cette guilde
	CanManage   bool   `json:"can_manage"` // l'utilisateur a MANAGE_GUILD
}

// handleListGuilds retourne la liste des guildes Discord de l'utilisateur,
// enrichie des informations de gestion bot.
// GET /guilds
func (srv *Server) handleListGuilds(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())

	discordGuilds, err := srv.fetchGuilds(r.Context(), sess.AccessToken)
	if err != nil {
		slog.Error("handlers: fetchGuilds échoué",
			"user_id", sess.UserID, "err", err)
		http.Error(w, "guilds fetch failed", http.StatusBadGateway)
		return
	}

	// Guildes actives connues du bot.
	botGuilds, err := srv.guildRepo.ListActive(r.Context())
	if err != nil {
		slog.Error("handlers: ListActive échoué", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	botGuildSet := make(map[string]struct{}, len(botGuilds))
	for _, g := range botGuilds {
		botGuildSet[g.GuildID] = struct{}{}
	}

	result := make([]guildResponse, 0, len(discordGuilds))
	for _, dg := range discordGuilds {
		_, managed := botGuildSet[dg.ID]
		perm, _ := strconv.ParseInt(dg.Permissions, 10, 64)
		result = append(result, guildResponse{
			ID:        dg.ID,
			Name:      dg.Name,
			Icon:      dg.Icon,
			Owner:     dg.Owner,
			Managed:   managed,
			CanManage: dg.Owner || (perm&manageGuildPermission != 0),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleGetGuild retourne le détail d'une guilde connue du bot.
// GET /guilds/{guildID}
func (srv *Server) handleGetGuild(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")

	g, err := srv.guildRepo.Get(r.Context(), guildID)
	if err != nil {
		slog.Error("handlers: guildRepo.Get échoué",
			"guild_id", guildID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if g == nil {
		http.Error(w, "guild not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(g)
}

// handleInstallBot génère l'URL d'installation du bot sur une guilde et redirige.
// POST /guilds/{guildID}/install
func (srv *Server) handleInstallBot(w http.ResponseWriter, r *http.Request) {
	guildID := chi.URLParam(r, "guildID")

	// Bitmask : VIEW_CHANNEL + SEND_MESSAGES + MANAGE_MESSAGES +
	//           KICK_MEMBERS + BAN_MEMBERS + MODERATE_MEMBERS
	const permissions = 1024 + 2048 + 8192 + 2 + 4 + 1099511627776

	params := url.Values{}
	params.Set("client_id", srv.cfg.DiscordClientID)
	params.Set("scope", "bot")
	params.Set("permissions", fmt.Sprintf("%d", permissions))
	params.Set("guild_id", guildID)
	params.Set("disable_guild_select", "true")

	installURL := "https://discord.com/api/oauth2/authorize?" + params.Encode()
	http.Redirect(w, r, installURL, http.StatusFound)
}

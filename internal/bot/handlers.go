// Package bot gère les handlers Gateway Discord.
package bot

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/repository"
)

// Handler reçoit les événements Discord et orchestre les réponses.
type Handler struct {
	guildRepo   repository.GuildRepository
	moduleRepo  repository.ModuleRepository
	configCache *cache.GuildConfigCache
}

// NewHandler crée un Handler.
func NewHandler(gr repository.GuildRepository, mr repository.ModuleRepository, cc *cache.GuildConfigCache) *Handler {
	return &Handler{
		guildRepo:   gr,
		moduleRepo:  mr,
		configCache: cc,
	}
}

// --- Handlers Gateway (wrappers privés pour discordgo) ---

func (h *Handler) onReady(s *discordgo.Session, r *discordgo.Ready) {
	slog.Info("bot: READY",
		"username", r.User.Username,
		"guilds", len(r.Guilds),
	)
}

func (h *Handler) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	h.HandleGuildCreate(context.Background(), g.Guild)
}

func (h *Handler) onGuildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	h.HandleGuildDelete(context.Background(), g.Guild)
}

// --- Méthodes exportées (testées directement) ---

// HandleGuildCreate persiste la guilde et pré-popule le cache de config.
func (h *Handler) HandleGuildCreate(ctx context.Context, g *discordgo.Guild) {
	if err := h.guildRepo.Upsert(ctx, repository.Guild{
		GuildID:   g.ID,
		GuildName: g.Name,
		Active:    true,
	}); err != nil {
		slog.Error("bot: GUILD_CREATE — upsert guilde échoué",
			"guild_id", g.ID, "err", err)
		return
	}
	slog.Info("bot: GUILD_CREATE — guilde persistée",
		"guild_id", g.ID, "guild_name", g.Name)

	// Pré-population du cache : échec non bloquant.
	if err := h.configCache.Populate(ctx, g.ID); err != nil {
		slog.Warn("bot: GUILD_CREATE — cache populate échoué",
			"guild_id", g.ID, "err", err)
	}
}

// HandleGuildDelete désactive la guilde et invalide le cache.
func (h *Handler) HandleGuildDelete(ctx context.Context, g *discordgo.Guild) {
	if err := h.guildRepo.Deactivate(ctx, g.ID); err != nil {
		slog.Error("bot: GUILD_DELETE — deactivate échoué",
			"guild_id", g.ID, "err", err)
	}

	// Invalidation du cache même si Deactivate a échoué.
	h.configCache.Invalidate(g.ID)
	slog.Info("bot: GUILD_DELETE — guilde désactivée",
		"guild_id", g.ID)
}

package bot

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// Handler reçoit les événements Gateway et les délègue aux couches métier.
type Handler struct {
	guildRepo repository.GuildRepository
}

// NewHandler crée un Handler avec ses dépendances.
func NewHandler(gr repository.GuildRepository) *Handler {
	return &Handler{guildRepo: gr}
}

// onReady est appelé quand Discord confirme l'authentification du bot.
func (h *Handler) onReady(s *discordgo.Session, r *discordgo.Ready) {
	slog.Info("bot: READY",
		"username", r.User.Username,
		"user_id", r.User.ID,
		"guilds", len(r.Guilds),
		"session_id", r.SessionID,
	)
}

// HandleGuildCreate est appelé pour chaque GUILD_CREATE (burst au démarrage
// et rejoint d'une nouvelle guilde en live). Exporté pour les tests.
func (h *Handler) HandleGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	ctx := context.Background()

	if err := h.guildRepo.Upsert(ctx, repository.Guild{
		GuildID:     g.ID,
		GuildName:   g.Name,
		OwnerUserID: g.OwnerID,
		Active:      true,
	}); err != nil {
		slog.Error("bot: GUILD_CREATE — upsert échoué",
			"guild_id", g.ID,
			"guild_name", g.Name,
			"err", err,
		)
		return
	}

	slog.Info("bot: GUILD_CREATE — guilde persistée",
		"guild_id", g.ID,
		"guild_name", g.Name,
	)
}

// HandleGuildDelete est appelé quand le bot est retiré d'une guilde ou qu'elle est supprimée.
// Exporté pour les tests.
func (h *Handler) HandleGuildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	ctx := context.Background()

	if err := h.guildRepo.Deactivate(ctx, g.ID); err != nil {
		slog.Error("bot: GUILD_DELETE — deactivate échoué",
			"guild_id", g.ID,
			"err", err,
		)
		return
	}

	slog.Info("bot: GUILD_DELETE — guilde désactivée", "guild_id", g.ID)
}

// onGuildCreate est le wrapper privé enregistré sur la session discordgo.
func (h *Handler) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	h.HandleGuildCreate(s, g)
}

// onGuildDelete est le wrapper privé enregistré sur la session discordgo.
func (h *Handler) onGuildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	h.HandleGuildDelete(s, g)
}

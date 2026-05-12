package module

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

// Dispatcher reçoit les événements Discord et les route vers les modules actifs.
type Dispatcher struct {
	registry    *Registry
	configCache *cache.GuildConfigCache
}

// NewDispatcher crée un Dispatcher.
func NewDispatcher(reg *Registry, cc *cache.GuildConfigCache) *Dispatcher {
	return &Dispatcher{registry: reg, configCache: cc}
}

// OnMessageCreate est le handler discordgo pour MESSAGE_CREATE.
// Il est enregistré sur la session dans bot.New().
func (d *Dispatcher) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignorer les messages du bot lui-même.
	if m.Author == nil || m.Author.Bot {
		return
	}
	// Ignorer les messages hors guilde (DM).
	if m.GuildID == "" {
		return
	}

	ctx := context.Background()

	cfg, err := d.configCache.Get(ctx, m.GuildID)
	if err != nil {
		slog.Error("dispatcher: cache.Get échoué",
			"guild_id", m.GuildID, "err", err)
		return
	}

	for _, mod := range d.registry.All() {
		// Vérifier l'activation du module pour cette guilde.
		if !cfg.IsEnabled(mod.Name()) {
			continue
		}
		if err := mod.HandleMessage(ctx, s, m, cfg); err != nil {
			slog.Warn("dispatcher: module error",
				"module", mod.Name(),
				"guild_id", m.GuildID,
				"channel_id", m.ChannelID,
				"author", m.Author.ID,
				"err", err,
			)
		}
	}
}

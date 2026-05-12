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
func (d *Dispatcher) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}
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

// OnGuildMemberUpdate est le handler discordgo pour GUILD_MEMBER_UPDATE.
// Il route l'événement vers les modules qui implémentent MemberUpdateHandler.
func (d *Dispatcher) OnGuildMemberUpdate(s *discordgo.Session, ev *discordgo.GuildMemberUpdate) {
	if ev.User == nil {
		return
	}
	ctx := context.Background()
	cfg, err := d.configCache.Get(ctx, ev.GuildID)
	if err != nil {
		slog.Error("dispatcher: cache.Get échoué (member_update)",
			"guild_id", ev.GuildID, "err", err)
		return
	}
	for _, mod := range d.registry.All() {
		h, ok := mod.(MemberUpdateHandler)
		if !ok {
			continue
		}
		if !cfg.IsEnabled(mod.Name()) {
			continue
		}
		if err := h.HandleMemberUpdate(ctx, s, ev, cfg); err != nil {
			slog.Warn("dispatcher: member_update error",
				"module", mod.Name(),
				"guild_id", ev.GuildID,
				"user_id", ev.User.ID,
				"err", err,
			)
		}
	}
}

// OnUserUpdate est le handler discordgo pour USER_UPDATE.
// USER_UPDATE est global (sans guild_id) : le dispatcher réplique l'événement
// dans toutes les guildes actives où le module est activé.
func (d *Dispatcher) OnUserUpdate(s *discordgo.Session, ev *discordgo.UserUpdate) {
	if ev.User == nil {
		return
	}
	ctx := context.Background()
	guildIDs := d.configCache.ActiveGuildIDs()
	for _, guildID := range guildIDs {
		cfg, err := d.configCache.Get(ctx, guildID)
		if err != nil {
			slog.Warn("dispatcher: cache.Get échoué (user_update)",
				"guild_id", guildID, "err", err)
			continue
		}
		for _, mod := range d.registry.All() {
			h, ok := mod.(UserUpdateHandler)
			if !ok {
				continue
			}
			if !cfg.IsEnabled(mod.Name()) {
				continue
			}
			if err := h.HandleUserUpdate(ctx, s, ev, guildID, cfg); err != nil {
				slog.Warn("dispatcher: user_update error",
					"module", mod.Name(),
					"guild_id", guildID,
					"user_id", ev.ID,
					"err", err,
				)
			}
		}
	}
}

package identityhistory

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

const ModuleName = "identity_history"

// IdentityHistory est le module de suivi des changements d'identité des membres.
type IdentityHistory struct {
	repo IdentityRepository
}

// New crée un nouveau module IdentityHistory.
func New(repo IdentityRepository) *IdentityHistory {
	return &IdentityHistory{repo: repo}
}

func (h *IdentityHistory) Name() string { return ModuleName }

// HandleMessage implémente module.Module mais identity_history n'agit pas sur les messages.
// Son événement principal est GUILD_MEMBER_UPDATE, enregistré séparément dans session.go.
func (h *IdentityHistory) HandleMessage(
	ctx context.Context,
	s *discordgo.Session,
	m *discordgo.MessageCreate,
	cfg *cache.GuildConfig,
) error {
	return nil
}

// HandleMemberUpdate est appelé sur chaque GUILD_MEMBER_UPDATE.
// Il compare l'ancienne et la nouvelle valeur de chaque champ configuré
// et insère un enregistrement si une différence est détectée.
func (h *IdentityHistory) HandleMemberUpdate(
	ctx context.Context,
	s *discordgo.Session,
	ev *discordgo.GuildMemberUpdate,
	cfg *cache.GuildConfig,
) error {
	var modCfg Config
	if err := cfg.ModuleConfig(ModuleName, &modCfg); err != nil {
		return err
	}
	modCfg.defaults()

	guildID := ev.GuildID
	userID := ev.User.ID

	type fieldCheck struct {
		enabled  bool
		field    FieldKind
		newValue string
	}

	// Construire le username pleinement qualifié (username#discriminator si non-zero).
	username := ev.User.Username
	if ev.User.Discriminator != "" && ev.User.Discriminator != "0" {
		username = ev.User.Username + "#" + ev.User.Discriminator
	}

	checks := []fieldCheck{
		{modCfg.TrackUsername, FieldUsername, username},
		{modCfg.TrackDisplayName, FieldDisplayName, ev.User.GlobalName},
		{modCfg.TrackNickname, FieldNickname, ev.Nick},
		{modCfg.TrackAvatar, FieldAvatar, ev.User.Avatar},
		{modCfg.TrackGuildAvatar, FieldGuildAvatar, ev.Avatar},
	}

	for _, c := range checks {
		if !c.enabled {
			continue
		}
		oldVal, err := h.repo.LastValue(ctx, guildID, userID, c.field)
		if err != nil {
			slog.Warn("identity_history: lecture dernière valeur échouée",
				"guild_id", guildID, "user_id", userID,
				"field", c.field, "err", err)
			continue
		}
		if oldVal == c.newValue {
			continue
		}
		rec := IdentityRecord{
			GuildID:  guildID,
			UserID:   userID,
			Field:    c.field,
			OldValue: oldVal,
			NewValue: c.newValue,
		}
		if err := h.repo.Insert(ctx, rec); err != nil {
			slog.Error("identity_history: insertion échouée",
				"guild_id", guildID, "user_id", userID,
				"field", c.field, "err", err)
			continue
		}
		slog.Info("identity_history: changement enregistré",
			"guild_id", guildID,
			"user_id", userID,
			"field", c.field,
			"old", oldVal,
			"new", c.newValue,
		)
	}
	return nil
}

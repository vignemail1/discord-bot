package identityhistory

import (
	"context"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

const ModuleName = "identity_history"

// fieldCheck représente un champ à contrôler lors d'un événement d'identité.
type fieldCheck struct {
	enabled  bool
	field    FieldKind
	newValue string
}

// IdentityHistory est le module de suivi des changements d'identité des membres.
type IdentityHistory struct {
	repo IdentityRepository
}

// New crée un nouveau module IdentityHistory.
func New(repo IdentityRepository) *IdentityHistory {
	return &IdentityHistory{repo: repo}
}

func (h *IdentityHistory) Name() string { return ModuleName }

// HandleMessage satisfait l'interface module.Module ; identity_history n'agit pas sur les messages.
func (h *IdentityHistory) HandleMessage(
	_ context.Context,
	_ *discordgo.Session,
	_ *discordgo.MessageCreate,
	_ *cache.GuildConfig,
) error {
	return nil
}

// HandleMemberUpdate implémente module.MemberUpdateHandler.
// Appelé sur GUILD_MEMBER_UPDATE : compare nick et avatar de guilde.
func (h *IdentityHistory) HandleMemberUpdate(
	ctx context.Context,
	_ *discordgo.Session,
	ev *discordgo.GuildMemberUpdate,
	cfg *cache.GuildConfig,
) error {
	var modCfg Config
	if err := cfg.ModuleConfig(ModuleName, &modCfg); err != nil {
		return err
	}
	modCfg.defaults()

	checks := []fieldCheck{
		{modCfg.TrackNickname, FieldNickname, ev.Nick},
		{modCfg.TrackGuildAvatar, FieldGuildAvatar, ev.Avatar},
	}

	return h.applyChecks(ctx, ev.GuildID, ev.User.ID, checks, "GUILD_MEMBER_UPDATE")
}

// HandleUserUpdate implémente module.UserUpdateHandler.
// Appelé sur USER_UPDATE (global) : compare username, display_name et avatar global.
// Le dispatcher appelle cette méthode pour chaque guilde active où le module est activé.
func (h *IdentityHistory) HandleUserUpdate(
	ctx context.Context,
	_ *discordgo.Session,
	ev *discordgo.UserUpdate,
	guildID string,
	cfg *cache.GuildConfig,
) error {
	var modCfg Config
	if err := cfg.ModuleConfig(ModuleName, &modCfg); err != nil {
		return err
	}
	modCfg.defaults()

	// Construire le username pleinement qualifié (username#discriminator si non-zero).
	username := ev.Username
	if ev.Discriminator != "" && ev.Discriminator != "0" {
		username = ev.Username + "#" + ev.Discriminator
	}

	checks := []fieldCheck{
		{modCfg.TrackUsername, FieldUsername, username},
		{modCfg.TrackDisplayName, FieldDisplayName, ev.GlobalName},
		{modCfg.TrackAvatar, FieldAvatar, ev.Avatar},
	}

	return h.applyChecks(ctx, guildID, ev.ID, checks, "USER_UPDATE")
}

// applyChecks compare les nouvelles valeurs aux dernières connues et insère les changements.
func (h *IdentityHistory) applyChecks(
	ctx context.Context,
	guildID, userID string,
	checks []fieldCheck,
	source string,
) error {
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
			GuildID:     guildID,
			UserID:      userID,
			Field:       c.field,
			OldValue:    oldVal,
			NewValue:    c.newValue,
			SourceEvent: source,
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
			"source", source,
		)
	}
	return nil
}

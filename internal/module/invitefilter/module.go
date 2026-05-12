package invitefilter

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

const ModuleName = "invite_filter"

// InviteFilter est le module de filtrage des liens d'invitation Discord.
type InviteFilter struct {
	counters CounterRepository
}

// New crée un nouveau module InviteFilter.
func New(counters CounterRepository) *InviteFilter {
	return &InviteFilter{counters: counters}
}

func (f *InviteFilter) Name() string { return ModuleName }

// HandleMessage est appelé par le Dispatcher pour chaque message sur une guilde active.
func (f *InviteFilter) HandleMessage(
	ctx context.Context,
	s *discordgo.Session,
	m *discordgo.MessageCreate,
	cfg *cache.GuildConfig,
) error {
	// Récupérer la config du module.
	var modCfg Config
	if err := cfg.ModuleConfig(ModuleName, &modCfg); err != nil {
		return err
	}
	modCfg.defaults()

	// Whitelist utilisateur.
	if slices.Contains(modCfg.WhitelistUserIDs, m.Author.ID) {
		return nil
	}
	// Whitelist rôle.
	if f.authorHasWhitelistedRole(m.Member, modCfg.WhitelistRoleIDs) {
		return nil
	}

	// Détecter les codes d'invitation.
	codes := ExtractInviteCodes(m.Content)
	if len(codes) == 0 {
		return nil
	}

	// Vérifier si tous les codes sont autorisés.
	if f.allCodesAllowed(codes, m.GuildID, modCfg) {
		return nil
	}

	// Supprimer le message.
	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		slog.Warn("invite_filter: suppression message échouée",
			"guild_id", m.GuildID, "channel_id", m.ChannelID,
			"msg_id", m.ID, "err", err)
	}

	// Incrémenter le compteur.
	count, err := f.counters.Increment(ctx, m.GuildID, m.Author.ID, ModuleName)
	if err != nil {
		slog.Error("invite_filter: incrément compteur échoué",
			"guild_id", m.GuildID, "user_id", m.Author.ID, "err", err)
		return err
	}

	slog.Info("invite_filter: lien interdit supprimé",
		"guild_id", m.GuildID,
		"user_id", m.Author.ID,
		"count", count,
		"codes", codes,
	)

	switch {
	case count >= modCfg.BanThreshold:
		f.ban(ctx, s, m, count)
	case count == 2:
		f.timeout(ctx, s, m, modCfg.TimeoutDuration)
	// count == 1 : suppression seule, déjà effectuée.
	}

	return nil
}

// --- actions de sanction ---

func (f *InviteFilter) timeout(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, durationStr string) {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		d = 24 * time.Hour
	}
	until := time.Now().Add(d)
	data := &discordgo.GuildMemberParams{
		CommunicationDisabledUntil: &until,
	}
	if _, err := s.GuildMemberEdit(m.GuildID, m.Author.ID, data); err != nil {
		slog.Error("invite_filter: timeout échoué",
			"guild_id", m.GuildID, "user_id", m.Author.ID, "err", err)
		return
	}
	slog.Info("invite_filter: timeout appliqué",
		"guild_id", m.GuildID, "user_id", m.Author.ID, "until", until)
}

func (f *InviteFilter) ban(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, count int) {
	if err := s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "invite_filter: liens interdits répétés", 0); err != nil {
		slog.Error("invite_filter: ban échoué",
			"guild_id", m.GuildID, "user_id", m.Author.ID, "err", err)
		return
	}
	slog.Info("invite_filter: ban appliqué",
		"guild_id", m.GuildID, "user_id", m.Author.ID, "count", count)
	// Reset du compteur après ban.
	_ = f.counters.Reset(ctx, m.GuildID, m.Author.ID, ModuleName)
}

// --- helpers ---

func (f *InviteFilter) authorHasWhitelistedRole(member *discordgo.Member, roleIDs []string) bool {
	if member == nil {
		return false
	}
	for _, r := range member.Roles {
		if slices.Contains(roleIDs, r) {
			return true
		}
	}
	return false
}

func (f *InviteFilter) allCodesAllowed(codes []string, guildID string, cfg Config) bool {
	for _, code := range codes {
		if IsAllowedCode(code, cfg.AllowedInviteCodes) {
			continue
		}
		// On ne peut pas résoudre le guild_id cible sans appel API Discord ;
		// si AllowedGuildIDs est vide, on bloque tout code non explicitement autorisé.
		// (La résolution API sera ajoutée lors du hardening step 11.)
		return false
	}
	return true
}

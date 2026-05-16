package invitefilter

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

const ModuleName = "invite_filter"

// InviteFilter est le module de filtrage des liens d'invitation Discord.
type InviteFilter struct {
	counters CounterRepository
	audit    AuditRepository
}

// New crée un nouveau module InviteFilter sans audit (rétrocompatibilité).
func New(counters CounterRepository) *InviteFilter {
	return &InviteFilter{counters: counters}
}

// NewWithAudit crée un module InviteFilter avec persistance de l'audit.
func NewWithAudit(counters CounterRepository, audit AuditRepository) *InviteFilter {
	return &InviteFilter{counters: counters, audit: audit}
}

func (f *InviteFilter) Name() string { return ModuleName }

// HandleMessage est appelé par le Dispatcher pour chaque message sur une guilde active.
func (f *InviteFilter) HandleMessage(
	ctx context.Context,
	s *discordgo.Session,
	m *discordgo.MessageCreate,
	cfg *cache.GuildConfig,
) error {
	var modCfg Config
	if err := cfg.ModuleConfig(ModuleName, &modCfg); err != nil {
		return err
	}
	modCfg.defaults()

	if slices.Contains(modCfg.WhitelistUserIDs, m.Author.ID) {
		return nil
	}
	if f.authorHasWhitelistedRole(m.Member, modCfg.WhitelistRoleIDs) {
		return nil
	}

	codes := ExtractInviteCodes(m.Content)
	if len(codes) == 0 {
		return nil
	}

	if f.allCodesAllowed(codes, m.GuildID, modCfg) {
		return nil
	}

	// Supprimer le message (skip si session non disponible, ex. en test unitaire).
	if s != nil {
		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			slog.Warn("invite_filter: suppression message échouée",
				"guild_id", m.GuildID, "channel_id", m.ChannelID,
				"msg_id", m.ID, "err", err)
		}
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

	var action ActionKind
	switch {
	case count >= modCfg.BanThreshold:
		f.ban(ctx, s, m, count)
		action = ActionBan
	case count == 2:
		f.timeout(ctx, s, m, modCfg.TimeoutDuration)
		action = ActionTimeout
	default:
		action = ActionDelete
	}

	f.writeAudit(ctx, m, codes, action, count)
	NotifyAction(ctx, s, modCfg, m, codes, action, count)

	return nil
}

// writeAudit persiste l'action si un AuditRepository est configuré.
func (f *InviteFilter) writeAudit(
	ctx context.Context,
	m *discordgo.MessageCreate,
	codes []string,
	action ActionKind,
	count int,
) {
	if f.audit == nil {
		return
	}
	rec := AuditRecord{
		GuildID:     m.GuildID,
		UserID:      m.Author.ID,
		ChannelID:   m.ChannelID,
		MessageID:   m.ID,
		Action:      action,
		InviteCodes: strings.Join(codes, ","),
		Count:       count,
	}
	if err := f.audit.Insert(ctx, rec); err != nil {
		slog.Error("invite_filter: audit insert échoué",
			"guild_id", m.GuildID, "user_id", m.Author.ID, "err", err)
	}
}

// --- actions de sanction ---

func (f *InviteFilter) timeout(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, durationStr string) {
	if s == nil {
		return
	}
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
	if s == nil {
		_ = f.counters.Reset(ctx, m.GuildID, m.Author.ID, ModuleName)
		return
	}
	if err := s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, "invite_filter: liens interdits répétés", 0); err != nil {
		slog.Error("invite_filter: ban échoué",
			"guild_id", m.GuildID, "user_id", m.Author.ID, "err", err)
		return
	}
	slog.Info("invite_filter: ban appliqué",
		"guild_id", m.GuildID, "user_id", m.Author.ID, "count", count)
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
		return false
	}
	return true
}

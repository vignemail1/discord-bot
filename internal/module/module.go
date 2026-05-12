// Package module définit le contrat commun à tous les modules de modération.
//
// Un module est une unité fonctionnelle indépendante qui reçoit les événements
// Discord filtrés par le Dispatcher. Chaque module définit son propre nom
// (utilisé comme clé dans guild_modules) et son comportement.
package module

import (
	"context"

	"github.com/bwmarrin/discordgo"

	"github.com/vignemail1/discord-bot/internal/cache"
)

// Module est le contrat implémenté par chaque module de modération.
type Module interface {
	// Name retourne le nom unique du module (ex : "invite_filter").
	// Doit correspondre exactement au module_name en base de données.
	Name() string

	// HandleMessage est appelé par le Dispatcher pour chaque message reçu
	// sur une guilde où ce module est actif.
	// cfg fournit l'accès à la config en cache (IsEnabled, ModuleConfig).
	// Retourner une erreur logue un warning sans interrompre les autres modules.
	HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, cfg *cache.GuildConfig) error
}

// MemberUpdateHandler est implémenté par les modules qui consomment GUILD_MEMBER_UPDATE.
type MemberUpdateHandler interface {
	HandleMemberUpdate(ctx context.Context, s *discordgo.Session, ev *discordgo.GuildMemberUpdate, cfg *cache.GuildConfig) error
}

// UserUpdateHandler est implémenté par les modules qui consomment USER_UPDATE.
// USER_UPDATE est global (pas de guild_id) ; le dispatcher réplique l'appel
// pour chaque guilde active où ce module est activé.
type UserUpdateHandler interface {
	HandleUserUpdate(ctx context.Context, s *discordgo.Session, ev *discordgo.UserUpdate, guildID string, cfg *cache.GuildConfig) error
}

// HandlerFunc permet d'enregistrer une fonction comme Module sans définir un type dédié.
// Utile pour les tests ou les modules one-shot simples.
type HandlerFunc struct {
	name string
	fn   func(context.Context, *discordgo.Session, *discordgo.MessageCreate, *cache.GuildConfig) error
}

// NewHandlerFunc crée un Module à partir d'une fonction.
func NewHandlerFunc(name string, fn func(context.Context, *discordgo.Session, *discordgo.MessageCreate, *cache.GuildConfig) error) *HandlerFunc {
	return &HandlerFunc{name: name, fn: fn}
}

func (h *HandlerFunc) Name() string { return h.name }
func (h *HandlerFunc) HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, cfg *cache.GuildConfig) error {
	return h.fn(ctx, s, m, cfg)
}

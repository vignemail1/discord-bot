package repository

import (
	"context"
	"time"
)

// IdentityState est l'état courant connu d'un membre (table guild_member_identity_state).
type IdentityState struct {
	GuildID        string
	UserID         string
	Username       *string
	GlobalName     *string
	GuildNick      *string
	AvatarHash     *string
	GuildAvatarHash *string
	FirstSeenAt    time.Time
	LastSeenAt     time.Time
}

// IdentityEvent est un événement de changement d'identité (table guild_member_identity_events).
type IdentityEvent struct {
	ID          int64
	GuildID     string
	UserID      string
	EventType   string
	OldValue    *string
	NewValue    *string
	ChangedAt   time.Time
	SourceEvent string
}

// IdentityFilter regroupe les paramètres de filtrage / pagination pour ListEvents.
type IdentityFilter struct {
	// EventType filtre sur un type de changement (optionnel).
	EventType string
	// Before est le curseur de pagination : retourne les événements dont l'ID < Before.
	// 0 = pas de curseur (première page).
	Before int64
	// Limit est le nombre maximum d'événements retournés (1-200, défaut 50).
	Limit int
}

// IdentityRepository est le contrat de persistance pour l'historique d'identité.
type IdentityRepository interface {
	// ListMembers retourne les états courants de tous les membres connus d'une guilde.
	ListMembers(ctx context.Context, guildID string) ([]IdentityState, error)
	// GetMember retourne l'état courant d'un membre. Retourne (nil, nil) si inconnu.
	GetMember(ctx context.Context, guildID, userID string) (*IdentityState, error)
	// ListMemberEvents retourne l'historique des changements d'un membre (ordre décroissant d'ID).
	ListMemberEvents(ctx context.Context, guildID, userID string, f IdentityFilter) ([]IdentityEvent, error)
}

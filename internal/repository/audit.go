package repository

import (
	"context"
	"time"
)

// AuditEvent représente un événement d'identité issu de guild_member_identity_events.
type AuditEvent struct {
	ID           int64
	GuildID      string
	UserID       string
	EventType    string
	OldValue     *string
	NewValue     *string
	ChangedAt    time.Time
	SourceEvent  string
	MetadataJSON *string
}

// AuditFilter regroupe les paramètres de filtrage / pagination pour ListEvents.
type AuditFilter struct {
	// UserID filtre sur un membre spécifique (optionnel).
	UserID string
	// EventType filtre sur un type d'événement (optionnel).
	EventType string
	// Before est le curseur de pagination : retourne les événements dont l'ID < Before.
	// 0 = pas de curseur (première page).
	Before int64
	// Limit est le nombre maximum d'événements retournés (max 200).
	Limit int
}

// AuditRepository est le contrat de persistance pour les événements d'audit.
type AuditRepository interface {
	// ListEvents retourne les événements d'une guilde en ordre décroissant d'ID,
	// avec filtrage optionnel et pagination par curseur.
	ListEvents(ctx context.Context, guildID string, f AuditFilter) ([]AuditEvent, error)
}

// Package repository définit les interfaces et modèles de persistance.
package repository

import (
	"context"
	"time"
)

// Guild représente une guilde Discord connue du bot.
type Guild struct {
	GuildID     string
	GuildName   string
	OwnerUserID string
	BotJoinedAt time.Time
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GuildRepository est le contrat de persistance pour les guildes.
type GuildRepository interface {
	// Upsert insère ou met à jour une guilde (guild_name, owner_user_id, active).
	Upsert(ctx context.Context, g Guild) error
	// Deactivate marque la guilde comme inactive (bot retiré ou guilde supprimée).
	Deactivate(ctx context.Context, guildID string) error
	// Get retourne la guilde par son ID. Retourne (nil, nil) si introuvable.
	Get(ctx context.Context, guildID string) (*Guild, error)
	// ListActive retourne toutes les guildes actives.
	ListActive(ctx context.Context) ([]Guild, error)
}

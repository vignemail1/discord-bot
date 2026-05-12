package repository

import (
	"context"
	"encoding/json"
	"time"
)

// GuildModule représente l'état d'un module pour une guilde.
type GuildModule struct {
	ID         int64
	GuildID    string
	ModuleName string
	Enabled    bool
	ConfigJSON json.RawMessage
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ModuleRepository est le contrat de persistance pour les modules par guilde.
type ModuleRepository interface {
	// Get retourne l'état d'un module pour une guilde. Retourne (nil, nil) si absent.
	Get(ctx context.Context, guildID, moduleName string) (*GuildModule, error)
	// ListByGuild retourne tous les modules enregistrés pour une guilde.
	ListByGuild(ctx context.Context, guildID string) ([]GuildModule, error)
	// Upsert insère ou met à jour l'état + config d'un module pour une guilde.
	Upsert(ctx context.Context, m GuildModule) error
	// SetEnabled active ou désactive un module sans toucher à la config.
	SetEnabled(ctx context.Context, guildID, moduleName string, enabled bool) error
	// UpdateConfig met à jour uniquement le config_json d'un module.
	UpdateConfig(ctx context.Context, guildID, moduleName string, config json.RawMessage) error
	// Delete supprime l'enregistrement d'un module pour une guilde.
	Delete(ctx context.Context, guildID, moduleName string) error
}

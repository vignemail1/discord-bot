// Package mariadb implémente les repositories avec MariaDB via sqlx.
package mariadb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// GuildRepo est l'implémentation MariaDB de repository.GuildRepository.
type GuildRepo struct {
	db *sqlx.DB
}

// NewGuildRepo crée un nouveau GuildRepo.
func NewGuildRepo(db *sqlx.DB) *GuildRepo {
	return &GuildRepo{db: db}
}

// Upsert insère ou met à jour une guilde.
// MariaDB INSERT ... ON DUPLICATE KEY UPDATE garantit l'atomicité.
func (r *GuildRepo) Upsert(ctx context.Context, g repository.Guild) error {
	const q = `
		INSERT INTO guilds (guild_id, guild_name, owner_user_id, bot_joined_at, active)
		VALUES (:guild_id, :guild_name, :owner_user_id, :bot_joined_at, :active)
		ON DUPLICATE KEY UPDATE
			guild_name    = VALUES(guild_name),
			owner_user_id = VALUES(owner_user_id),
			active        = VALUES(active)
	`

	now := time.Now().UTC()
	if g.BotJoinedAt.IsZero() {
		g.BotJoinedAt = now
	}

	params := map[string]any{
		"guild_id":      g.GuildID,
		"guild_name":    g.GuildName,
		"owner_user_id": g.OwnerUserID,
		"bot_joined_at": g.BotJoinedAt,
		"active":        g.Active,
	}

	_, err := r.db.NamedExecContext(ctx, q, params)
	if err != nil {
		return fmt.Errorf("guildRepo.Upsert: %w", err)
	}
	return nil
}

// Deactivate met active=0 pour la guilde donnée.
func (r *GuildRepo) Deactivate(ctx context.Context, guildID string) error {
	const q = `UPDATE guilds SET active = 0 WHERE guild_id = ?`
	_, err := r.db.ExecContext(ctx, q, guildID)
	if err != nil {
		return fmt.Errorf("guildRepo.Deactivate: %w", err)
	}
	return nil
}

// Get retourne une guilde par son ID. Retourne (nil, nil) si introuvable.
func (r *GuildRepo) Get(ctx context.Context, guildID string) (*repository.Guild, error) {
	const q = `
		SELECT guild_id, guild_name, owner_user_id, bot_joined_at, active, created_at, updated_at
		FROM guilds
		WHERE guild_id = ?
	`

	var row struct {
		GuildID     string    `db:"guild_id"`
		GuildName   string    `db:"guild_name"`
		OwnerUserID string    `db:"owner_user_id"`
		BotJoinedAt time.Time `db:"bot_joined_at"`
		Active      bool      `db:"active"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}

	if err := r.db.GetContext(ctx, &row, q, guildID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("guildRepo.Get: %w", err)
	}

	return &repository.Guild{
		GuildID:     row.GuildID,
		GuildName:   row.GuildName,
		OwnerUserID: row.OwnerUserID,
		BotJoinedAt: row.BotJoinedAt,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

// ListActive retourne toutes les guildes actives.
func (r *GuildRepo) ListActive(ctx context.Context) ([]repository.Guild, error) {
	const q = `
		SELECT guild_id, guild_name, owner_user_id, bot_joined_at, active, created_at, updated_at
		FROM guilds
		WHERE active = 1
		ORDER BY guild_name ASC
	`

	var rows []struct {
		GuildID     string    `db:"guild_id"`
		GuildName   string    `db:"guild_name"`
		OwnerUserID string    `db:"owner_user_id"`
		BotJoinedAt time.Time `db:"bot_joined_at"`
		Active      bool      `db:"active"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}

	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("guildRepo.ListActive: %w", err)
	}

	guilds := make([]repository.Guild, 0, len(rows))
	for _, row := range rows {
		guilds = append(guilds, repository.Guild{
			GuildID:     row.GuildID,
			GuildName:   row.GuildName,
			OwnerUserID: row.OwnerUserID,
			BotJoinedAt: row.BotJoinedAt,
			Active:      row.Active,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return guilds, nil
}

// Vérification statique : GuildRepo implémente repository.GuildRepository.
var _ repository.GuildRepository = (*GuildRepo)(nil)

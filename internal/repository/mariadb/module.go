package mariadb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// ModuleRepo est l'implémentation MariaDB de repository.ModuleRepository.
type ModuleRepo struct {
	db *sqlx.DB
}

// NewModuleRepo crée un nouveau ModuleRepo.
func NewModuleRepo(db *sqlx.DB) *ModuleRepo {
	return &ModuleRepo{db: db}
}

func (r *ModuleRepo) Get(ctx context.Context, guildID, moduleName string) (*repository.GuildModule, error) {
	const q = `
		SELECT id, guild_id, module_name, enabled, config_json, created_at, updated_at
		FROM guild_modules
		WHERE guild_id = ? AND module_name = ?
	`
	var row struct {
		ID         int64          `db:"id"`
		GuildID    string         `db:"guild_id"`
		ModuleName string         `db:"module_name"`
		Enabled    bool           `db:"enabled"`
		ConfigJSON []byte         `db:"config_json"`
		CreatedAt  sql.NullTime   `db:"created_at"`
		UpdatedAt  sql.NullTime   `db:"updated_at"`
	}
	if err := r.db.GetContext(ctx, &row, q, guildID, moduleName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("moduleRepo.Get: %w", err)
	}
	return &repository.GuildModule{
		ID:         row.ID,
		GuildID:    row.GuildID,
		ModuleName: row.ModuleName,
		Enabled:    row.Enabled,
		ConfigJSON: json.RawMessage(row.ConfigJSON),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *ModuleRepo) ListByGuild(ctx context.Context, guildID string) ([]repository.GuildModule, error) {
	const q = `
		SELECT id, guild_id, module_name, enabled, config_json, created_at, updated_at
		FROM guild_modules
		WHERE guild_id = ?
		ORDER BY module_name ASC
	`
	var rows []struct {
		ID         int64        `db:"id"`
		GuildID    string       `db:"guild_id"`
		ModuleName string       `db:"module_name"`
		Enabled    bool         `db:"enabled"`
		ConfigJSON []byte       `db:"config_json"`
		CreatedAt  sql.NullTime `db:"created_at"`
		UpdatedAt  sql.NullTime `db:"updated_at"`
	}
	if err := r.db.SelectContext(ctx, &rows, q, guildID); err != nil {
		return nil, fmt.Errorf("moduleRepo.ListByGuild: %w", err)
	}
	out := make([]repository.GuildModule, 0, len(rows))
	for _, row := range rows {
		out = append(out, repository.GuildModule{
			ID:         row.ID,
			GuildID:    row.GuildID,
			ModuleName: row.ModuleName,
			Enabled:    row.Enabled,
			ConfigJSON: json.RawMessage(row.ConfigJSON),
			CreatedAt:  row.CreatedAt.Time,
			UpdatedAt:  row.UpdatedAt.Time,
		})
	}
	return out, nil
}

func (r *ModuleRepo) Upsert(ctx context.Context, m repository.GuildModule) error {
	const q = `
		INSERT INTO guild_modules (guild_id, module_name, enabled, config_json)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			enabled     = VALUES(enabled),
			config_json = VALUES(config_json)
	`
	cfg := m.ConfigJSON
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	_, err := r.db.ExecContext(ctx, q, m.GuildID, m.ModuleName, m.Enabled, []byte(cfg))
	if err != nil {
		return fmt.Errorf("moduleRepo.Upsert: %w", err)
	}
	return nil
}

func (r *ModuleRepo) SetEnabled(ctx context.Context, guildID, moduleName string, enabled bool) error {
	const q = `UPDATE guild_modules SET enabled = ? WHERE guild_id = ? AND module_name = ?`
	_, err := r.db.ExecContext(ctx, q, enabled, guildID, moduleName)
	if err != nil {
		return fmt.Errorf("moduleRepo.SetEnabled: %w", err)
	}
	return nil
}

func (r *ModuleRepo) UpdateConfig(ctx context.Context, guildID, moduleName string, config json.RawMessage) error {
	const q = `UPDATE guild_modules SET config_json = ? WHERE guild_id = ? AND module_name = ?`
	_, err := r.db.ExecContext(ctx, q, []byte(config), guildID, moduleName)
	if err != nil {
		return fmt.Errorf("moduleRepo.UpdateConfig: %w", err)
	}
	return nil
}

func (r *ModuleRepo) Delete(ctx context.Context, guildID, moduleName string) error {
	const q = `DELETE FROM guild_modules WHERE guild_id = ? AND module_name = ?`
	_, err := r.db.ExecContext(ctx, q, guildID, moduleName)
	if err != nil {
		return fmt.Errorf("moduleRepo.Delete: %w", err)
	}
	return nil
}

// Vérification statique.
var _ repository.ModuleRepository = (*ModuleRepo)(nil)

package identityhistory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// FieldKind représente le type de champ qui a changé.
type FieldKind string

const (
	FieldUsername    FieldKind = "username"
	FieldDisplayName FieldKind = "display_name"
	FieldNickname    FieldKind = "nickname"
	FieldAvatar      FieldKind = "avatar"
	FieldGuildAvatar FieldKind = "guild_avatar"
)

// IdentityRecord est une entrée de l'historique d'identité.
type IdentityRecord struct {
	ID        int64     `db:"id"`
	GuildID   string    `db:"guild_id"`
	UserID    string    `db:"user_id"`
	Field     FieldKind `db:"field"`
	OldValue  string    `db:"old_value"`
	NewValue  string    `db:"new_value"`
	CreatedAt time.Time `db:"created_at"`
}

// IdentityRepository est le contrat de persistance de l'historique.
type IdentityRepository interface {
	// Insert enregistre un changement d'identité.
	Insert(ctx context.Context, r IdentityRecord) error
	// ListByUser retourne l'historique d'un utilisateur (du plus récent au plus ancien).
	ListByUser(ctx context.Context, guildID, userID string, limit int) ([]IdentityRecord, error)
	// Purge supprime les enregistrements antérieurs à before.
	Purge(ctx context.Context, guildID string, before time.Time) (int64, error)
	// LastValue retourne la dernière valeur connue d'un champ pour un utilisateur.
	// Retourne ("", nil) si aucun enregistrement n'existe.
	LastValue(ctx context.Context, guildID, userID string, field FieldKind) (string, error)
}

// MariaDBIdentityRepo est l'implémentation MariaDB de IdentityRepository.
type MariaDBIdentityRepo struct {
	db *sqlx.DB
}

// NewMariaDBIdentityRepo crée un nouveau MariaDBIdentityRepo.
func NewMariaDBIdentityRepo(db *sqlx.DB) *MariaDBIdentityRepo {
	return &MariaDBIdentityRepo{db: db}
}

func (r *MariaDBIdentityRepo) Insert(ctx context.Context, rec IdentityRecord) error {
	const q = `
		INSERT INTO guild_member_identity_history
			(guild_id, user_id, field, old_value, new_value)
		VALUES (?, ?, ?, ?, ?)
	`
	if _, err := r.db.ExecContext(ctx, q,
		rec.GuildID, rec.UserID, rec.Field, rec.OldValue, rec.NewValue,
	); err != nil {
		return fmt.Errorf("identity.Insert: %w", err)
	}
	return nil
}

func (r *MariaDBIdentityRepo) ListByUser(ctx context.Context, guildID, userID string, limit int) ([]IdentityRecord, error) {
	const q = `
		SELECT id, guild_id, user_id, field, old_value, new_value, created_at
		FROM guild_member_identity_history
		WHERE guild_id = ? AND user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	var out []IdentityRecord
	if err := r.db.SelectContext(ctx, &out, q, guildID, userID, limit); err != nil {
		return nil, fmt.Errorf("identity.ListByUser: %w", err)
	}
	return out, nil
}

func (r *MariaDBIdentityRepo) Purge(ctx context.Context, guildID string, before time.Time) (int64, error) {
	const q = `DELETE FROM guild_member_identity_history WHERE guild_id = ? AND created_at < ?`
	res, err := r.db.ExecContext(ctx, q, guildID, before)
	if err != nil {
		return 0, fmt.Errorf("identity.Purge: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (r *MariaDBIdentityRepo) LastValue(ctx context.Context, guildID, userID string, field FieldKind) (string, error) {
	const q = `
		SELECT new_value FROM guild_member_identity_history
		WHERE guild_id = ? AND user_id = ? AND field = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var val string
	if err := r.db.GetContext(ctx, &val, q, guildID, userID, field); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("identity.LastValue: %w", err)
	}
	return val, nil
}

// --- Implémentation mémoire pour les tests ---

// MemoryIdentityRepo est une implémentation en mémoire de IdentityRepository.
type MemoryIdentityRepo struct {
	Records   []IdentityRecord
	InsertErr error
}

func NewMemoryIdentityRepo() *MemoryIdentityRepo {
	return &MemoryIdentityRepo{}
}

func (r *MemoryIdentityRepo) Insert(ctx context.Context, rec IdentityRecord) error {
	if r.InsertErr != nil {
		return r.InsertErr
	}
	rec.ID = int64(len(r.Records) + 1)
	rec.CreatedAt = time.Now()
	r.Records = append(r.Records, rec)
	return nil
}

func (r *MemoryIdentityRepo) ListByUser(ctx context.Context, guildID, userID string, limit int) ([]IdentityRecord, error) {
	var out []IdentityRecord
	for i := len(r.Records) - 1; i >= 0; i-- {
		v := r.Records[i]
		if v.GuildID == guildID && v.UserID == userID {
			out = append(out, v)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryIdentityRepo) Purge(ctx context.Context, guildID string, before time.Time) (int64, error) {
	var kept []IdentityRecord
	var n int64
	for _, v := range r.Records {
		if v.GuildID == guildID && v.CreatedAt.Before(before) {
			n++
		} else {
			kept = append(kept, v)
		}
	}
	r.Records = kept
	return n, nil
}

func (r *MemoryIdentityRepo) LastValue(ctx context.Context, guildID, userID string, field FieldKind) (string, error) {
	for i := len(r.Records) - 1; i >= 0; i-- {
		v := r.Records[i]
		if v.GuildID == guildID && v.UserID == userID && v.Field == field {
			return v.NewValue, nil
		}
	}
	return "", nil
}

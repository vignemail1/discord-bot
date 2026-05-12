package invitefilter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ActionKind identifie le type de sanction appliquée.
type ActionKind string

const (
	ActionDelete  ActionKind = "delete"  // Message supprimé uniquement.
	ActionTimeout ActionKind = "timeout" // Timeout appliqué.
	ActionBan     ActionKind = "ban"     // Ban appliqué.
)

// AuditRecord est une entrée de l'audit de modération.
type AuditRecord struct {
	ID          int64      `db:"id"`
	GuildID     string     `db:"guild_id"`
	UserID      string     `db:"user_id"`
	ChannelID   string     `db:"channel_id"`
	MessageID   string     `db:"message_id"`
	Action      ActionKind `db:"action"`
	InviteCodes string     `db:"invite_codes"` // codes extraits, séparés par des virgules
	Count       int        `db:"counter_value"` // valeur du compteur au moment de l'action
	CreatedAt   time.Time  `db:"created_at"`
}

// AuditRepository est le contrat de persistance de l'audit invite_filter.
type AuditRepository interface {
	// Insert enregistre une action de modération.
	Insert(ctx context.Context, r AuditRecord) error
	// ListByUser retourne les actions d'un utilisateur (du plus récent au plus ancien).
	ListByUser(ctx context.Context, guildID, userID string, limit int) ([]AuditRecord, error)
	// ListByGuild retourne les dernières actions d'une guilde.
	ListByGuild(ctx context.Context, guildID string, limit int) ([]AuditRecord, error)
}

// MariaDBauditRepo est l'implémentation MariaDB de AuditRepository.
type MariaDBauditRepo struct {
	db *sqlx.DB
}

// NewMariaDBauditRepo crée un nouveau MariaDBauditRepo.
func NewMariaDBauditRepo(db *sqlx.DB) *MariaDBauditRepo {
	return &MariaDBauditRepo{db: db}
}

func (r *MariaDBauditRepo) Insert(ctx context.Context, rec AuditRecord) error {
	const q = `
		INSERT INTO invite_filter_audit
			(guild_id, user_id, channel_id, message_id, action, invite_codes, counter_value)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, q,
		rec.GuildID, rec.UserID, rec.ChannelID, rec.MessageID,
		rec.Action, rec.InviteCodes, rec.Count,
	)
	if err != nil {
		return fmt.Errorf("audit.Insert: %w", err)
	}
	return nil
}

func (r *MariaDBauditRepo) ListByUser(ctx context.Context, guildID, userID string, limit int) ([]AuditRecord, error) {
	const q = `
		SELECT id, guild_id, user_id, channel_id, message_id, action, invite_codes, counter_value, created_at
		FROM invite_filter_audit
		WHERE guild_id = ? AND user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	var out []AuditRecord
	if err := r.db.SelectContext(ctx, &out, q, guildID, userID, limit); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit.ListByUser: %w", err)
	}
	return out, nil
}

func (r *MariaDBauditRepo) ListByGuild(ctx context.Context, guildID string, limit int) ([]AuditRecord, error) {
	const q = `
		SELECT id, guild_id, user_id, channel_id, message_id, action, invite_codes, counter_value, created_at
		FROM invite_filter_audit
		WHERE guild_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	var out []AuditRecord
	if err := r.db.SelectContext(ctx, &out, q, guildID, limit); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit.ListByGuild: %w", err)
	}
	return out, nil
}

// --- Implémentation mémoire pour les tests ---

// MemoryAuditRepo est une implémentation en mémoire de AuditRepository.
type MemoryAuditRepo struct {
	Records   []AuditRecord
	InsertErr error
}

// NewMemoryAuditRepo crée un MemoryAuditRepo vide.
func NewMemoryAuditRepo() *MemoryAuditRepo {
	return &MemoryAuditRepo{}
}

func (r *MemoryAuditRepo) Insert(ctx context.Context, rec AuditRecord) error {
	if r.InsertErr != nil {
		return r.InsertErr
	}
	rec.ID = int64(len(r.Records) + 1)
	rec.CreatedAt = time.Now()
	r.Records = append(r.Records, rec)
	return nil
}

func (r *MemoryAuditRepo) ListByUser(ctx context.Context, guildID, userID string, limit int) ([]AuditRecord, error) {
	var out []AuditRecord
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

func (r *MemoryAuditRepo) ListByGuild(ctx context.Context, guildID string, limit int) ([]AuditRecord, error) {
	var out []AuditRecord
	for i := len(r.Records) - 1; i >= 0; i-- {
		v := r.Records[i]
		if v.GuildID == guildID {
			out = append(out, v)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

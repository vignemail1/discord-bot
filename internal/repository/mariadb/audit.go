package mariadb

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// AuditRepo est l'implémentation MariaDB de repository.AuditRepository.
type AuditRepo struct {
	db *sqlx.DB
}

func NewAuditRepo(db *sqlx.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) ListEvents(ctx context.Context, guildID string, f repository.AuditFilter) ([]repository.AuditEvent, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var sb strings.Builder
	args := make([]interface{}, 0, 5)

	sb.WriteString(`
		SELECT id, guild_id, user_id, event_type,
		       old_value, new_value, changed_at,
		       source_event, metadata_json
		FROM guild_member_identity_events
		WHERE guild_id = ?`)
	args = append(args, guildID)

	if f.UserID != "" {
		sb.WriteString(" AND user_id = ?")
		args = append(args, f.UserID)
	}
	if f.EventType != "" {
		sb.WriteString(" AND event_type = ?")
		args = append(args, f.EventType)
	}
	if f.Before > 0 {
		sb.WriteString(" AND id < ?")
		args = append(args, f.Before)
	}

	sb.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit)

	type row struct {
		ID           int64   `db:"id"`
		GuildID      string  `db:"guild_id"`
		UserID       string  `db:"user_id"`
		EventType    string  `db:"event_type"`
		OldValue     *string `db:"old_value"`
		NewValue     *string `db:"new_value"`
		ChangedAt    string  `db:"changed_at"`
		SourceEvent  string  `db:"source_event"`
		MetadataJSON *string `db:"metadata_json"`
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, err
	}

	out := make([]repository.AuditEvent, 0, len(rows))
	for _, rw := range rows {
		changedAt, _ := parseDateTime(rw.ChangedAt)
		out = append(out, repository.AuditEvent{
			ID:           rw.ID,
			GuildID:      rw.GuildID,
			UserID:       rw.UserID,
			EventType:    rw.EventType,
			OldValue:     rw.OldValue,
			NewValue:     rw.NewValue,
			ChangedAt:    changedAt,
			SourceEvent:  rw.SourceEvent,
			MetadataJSON: rw.MetadataJSON,
		})
	}
	return out, nil
}

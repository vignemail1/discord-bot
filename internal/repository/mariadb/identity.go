package mariadb

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiern/sqlx"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// IdentityRepo est l'implémentation MariaDB de repository.IdentityRepository.
type IdentityRepo struct {
	db *sqlx.DB
}

func NewIdentityRepo(db *sqlx.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

func (r *IdentityRepo) ListMembers(ctx context.Context, guildID string) ([]repository.IdentityState, error) {
	const q = `
		SELECT guild_id, user_id,
		       username, global_name, guild_nick,
		       avatar_hash, guild_avatar_hash,
		       first_seen_at, last_seen_at
		FROM guild_member_identity_state
		WHERE guild_id = ?
		ORDER BY last_seen_at DESC
	`
	type row struct {
		GuildID         string  `db:"guild_id"`
		UserID          string  `db:"user_id"`
		Username        *string `db:"username"`
		GlobalName      *string `db:"global_name"`
		GuildNick       *string `db:"guild_nick"`
		AvatarHash      *string `db:"avatar_hash"`
		GuildAvatarHash *string `db:"guild_avatar_hash"`
		FirstSeenAt     string  `db:"first_seen_at"`
		LastSeenAt      string  `db:"last_seen_at"`
	}
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, q, guildID); err != nil {
		return nil, err
	}
	out := make([]repository.IdentityState, 0, len(rows))
	for _, rw := range rows {
		first, _ := parseDateTime(rw.FirstSeenAt)
		last, _ := parseDateTime(rw.LastSeenAt)
		out = append(out, repository.IdentityState{
			GuildID: rw.GuildID, UserID: rw.UserID,
			Username: rw.Username, GlobalName: rw.GlobalName,
			GuildNick: rw.GuildNick, AvatarHash: rw.AvatarHash,
			GuildAvatarHash: rw.GuildAvatarHash,
			FirstSeenAt: first, LastSeenAt: last,
		})
	}
	return out, nil
}

func (r *IdentityRepo) GetMember(ctx context.Context, guildID, userID string) (*repository.IdentityState, error) {
	const q = `
		SELECT guild_id, user_id,
		       username, global_name, guild_nick,
		       avatar_hash, guild_avatar_hash,
		       first_seen_at, last_seen_at
		FROM guild_member_identity_state
		WHERE guild_id = ? AND user_id = ?
	`
	type row struct {
		GuildID         string  `db:"guild_id"`
		UserID          string  `db:"user_id"`
		Username        *string `db:"username"`
		GlobalName      *string `db:"global_name"`
		GuildNick       *string `db:"guild_nick"`
		AvatarHash      *string `db:"avatar_hash"`
		GuildAvatarHash *string `db:"guild_avatar_hash"`
		FirstSeenAt     string  `db:"first_seen_at"`
		LastSeenAt      string  `db:"last_seen_at"`
	}
	var rw row
	if err := r.db.GetContext(ctx, &rw, q, guildID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	first, _ := parseDateTime(rw.FirstSeenAt)
	last, _ := parseDateTime(rw.LastSeenAt)
	return &repository.IdentityState{
		GuildID: rw.GuildID, UserID: rw.UserID,
		Username: rw.Username, GlobalName: rw.GlobalName,
		GuildNick: rw.GuildNick, AvatarHash: rw.AvatarHash,
		GuildAvatarHash: rw.GuildAvatarHash,
		FirstSeenAt: first, LastSeenAt: last,
	}, nil
}

func (r *IdentityRepo) ListMemberEvents(ctx context.Context, guildID, userID string, f repository.IdentityFilter) ([]repository.IdentityEvent, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var sb strings.Builder
	args := make([]interface{}, 0, 5)

	sb.WriteString(`
		SELECT id, guild_id, user_id, event_type,
		       old_value, new_value, changed_at, source_event
		FROM guild_member_identity_events
		WHERE guild_id = ? AND user_id = ?`)
	args = append(args, guildID, userID)

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
		ID          int64   `db:"id"`
		GuildID     string  `db:"guild_id"`
		UserID      string  `db:"user_id"`
		EventType   string  `db:"event_type"`
		OldValue    *string `db:"old_value"`
		NewValue    *string `db:"new_value"`
		ChangedAt   string  `db:"changed_at"`
		SourceEvent string  `db:"source_event"`
	}
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, err
	}
	out := make([]repository.IdentityEvent, 0, len(rows))
	for _, rw := range rows {
		changedAt, _ := parseDateTime(rw.ChangedAt)
		out = append(out, repository.IdentityEvent{
			ID: rw.ID, GuildID: rw.GuildID, UserID: rw.UserID,
			EventType: rw.EventType, OldValue: rw.OldValue, NewValue: rw.NewValue,
			ChangedAt: changedAt, SourceEvent: rw.SourceEvent,
		})
	}
	return out, nil
}

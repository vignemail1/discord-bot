package mariadb_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mariadb"
)

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "mysql"), mock
}

func TestGuildRepo_Upsert(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO guilds`).
		WithArgs(
			sqlmock.AnyArg(), // guild_id
			sqlmock.AnyArg(), // guild_name
			sqlmock.AnyArg(), // owner_user_id
			sqlmock.AnyArg(), // bot_joined_at
			sqlmock.AnyArg(), // active
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := mariadb.NewGuildRepo(db)
	err := repo.Upsert(context.Background(), repository.Guild{
		GuildID:     "123",
		GuildName:   "Test Guild",
		OwnerUserID: "456",
		Active:      true,
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGuildRepo_Deactivate(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectExec(`UPDATE guilds SET active`).
		WithArgs("123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := mariadb.NewGuildRepo(db)
	err := repo.Deactivate(context.Background(), "123")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGuildRepo_Get_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	cols := []string{"guild_id", "guild_name", "owner_user_id", "bot_joined_at", "active", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT .+ FROM guilds WHERE guild_id`).
		WithArgs("999").
		WillReturnRows(sqlmock.NewRows(cols))

	repo := mariadb.NewGuildRepo(db)
	g, err := repo.Get(context.Background(), "999")

	require.NoError(t, err)
	assert.Nil(t, g)
}

func TestGuildRepo_Get_Found(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	cols := []string{"guild_id", "guild_name", "owner_user_id", "bot_joined_at", "active", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT .+ FROM guilds WHERE guild_id`).
		WithArgs("123").
		WillReturnRows(sqlmock.NewRows(cols).AddRow("123", "Test", "456", now, true, now, now))

	repo := mariadb.NewGuildRepo(db)
	g, err := repo.Get(context.Background(), "123")

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "123", g.GuildID)
	assert.Equal(t, "Test", g.GuildName)
	assert.True(t, g.Active)
}

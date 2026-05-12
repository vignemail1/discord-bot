package mariadb_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mariadb"
)

func TestModuleRepo_Upsert(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectExec(`INSERT INTO guild_modules`).
		WithArgs("111", "invite_filter", true, []byte(`{"foo":"bar"}`)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := mariadb.NewModuleRepo(db)
	err := repo.Upsert(context.Background(), repository.GuildModule{
		GuildID:    "111",
		ModuleName: "invite_filter",
		Enabled:    true,
		ConfigJSON: json.RawMessage(`{"foo":"bar"}`),
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestModuleRepo_Get_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	cols := []string{"id", "guild_id", "module_name", "enabled", "config_json", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT .+ FROM guild_modules WHERE guild_id`).
		WithArgs("111", "invite_filter").
		WillReturnRows(sqlmock.NewRows(cols))

	repo := mariadb.NewModuleRepo(db)
	m, err := repo.Get(context.Background(), "111", "invite_filter")

	require.NoError(t, err)
	assert.Nil(t, m)
}

func TestModuleRepo_Get_Found(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	now := time.Now().Truncate(time.Second)
	cols := []string{"id", "guild_id", "module_name", "enabled", "config_json", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT .+ FROM guild_modules WHERE guild_id`).
		WithArgs("111", "invite_filter").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(1, "111", "invite_filter", true, []byte(`{}`), now, now))

	repo := mariadb.NewModuleRepo(db)
	m, err := repo.Get(context.Background(), "111", "invite_filter")

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "invite_filter", m.ModuleName)
	assert.True(t, m.Enabled)
}

func TestModuleRepo_SetEnabled(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectExec(`UPDATE guild_modules SET enabled`).
		WithArgs(false, "111", "invite_filter").
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := mariadb.NewModuleRepo(db)
	err := repo.SetEnabled(context.Background(), "111", "invite_filter", false)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

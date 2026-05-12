package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	for _, k := range []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "WEB_LISTEN_ADDR", "LOG_LEVEL"} {
		t.Setenv(k, "")
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "mariadb", cfg.DBHost)
	assert.Equal(t, 3306, cfg.DBPort)
	assert.Equal(t, "discordbot", cfg.DBName)
	assert.Equal(t, "discordbot", cfg.DBUser)
	assert.Equal(t, ":8080", cfg.WebListenAddr)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_USER", "root")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 3307, cfg.DBPort)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("DB_PORT", "not-a-number")

	_, err := config.Load()
	assert.Error(t, err)
}

func TestDSN(t *testing.T) {
	os.Setenv("DB_HOST", "db")
	os.Setenv("DB_PORT", "3306")
	os.Setenv("DB_NAME", "mydb")
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASSWORD", "pass")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Contains(t, cfg.DSN(), "user:pass@tcp(db:3306)/mydb")
}

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	for _, k := range []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "HTTP_ADDR", "LOG_LEVEL", "CACHE_TTL_SECONDS"} {
		t.Setenv(k, "")
	}
	// CACHE_TTL_SECONDS vide provoque une erreur ; fournir une valeur valide.
	t.Setenv("CACHE_TTL_SECONDS", "300")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "mariadb", cfg.DBHost)
	assert.Equal(t, "3306", cfg.DBPort)
	assert.Equal(t, "discordbot", cfg.DBName)
	assert.Equal(t, "discordbot", cfg.DBUser)
	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_USER", "root")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("CACHE_TTL_SECONDS", "60")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "3307", cfg.DBPort)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidCacheTTL(t *testing.T) {
	t.Setenv("CACHE_TTL_SECONDS", "not-a-number")

	_, err := config.Load()
	assert.Error(t, err)
}

func TestDSN(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("CACHE_TTL_SECONDS", "300")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Contains(t, cfg.DSN(), "user:pass@tcp(db:3306)/mydb")
}

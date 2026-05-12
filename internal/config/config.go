// Package config charge la configuration depuis les variables d'environnement.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config contient toute la configuration du service.
type Config struct {
	// Bot Discord
	DiscordBotToken string
	DiscordClientID string
	DiscordClientSecret string
	DiscordRedirectURL string

	// Base de données MariaDB
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Dashboard web
	HTTPAddr      string
	SessionSecret string

	// Cache
	CacheTTL time.Duration

	// Logs
	LogLevel string
}

// Load lit les variables d'environnement et retourne une Config validée.
func Load() (*Config, error) {
	c := &Config{
		DiscordBotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURL:  getEnvDefault("DISCORD_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		DBHost:              getEnvDefault("DB_HOST", "mariadb"),
		DBPort:              getEnvDefault("DB_PORT", "3306"),
		DBUser:              getEnvDefault("DB_USER", "discordbot"),
		DBPassword:          os.Getenv("DB_PASSWORD"),
		DBName:              getEnvDefault("DB_NAME", "discordbot"),
		HTTPAddr:            getEnvDefault("HTTP_ADDR", ":8080"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		LogLevel:            getEnvDefault("LOG_LEVEL", "info"),
	}

	ttlSec, err := strconv.Atoi(getEnvDefault("CACHE_TTL_SECONDS", "300"))
	if err != nil || ttlSec <= 0 {
		return nil, fmt.Errorf("config: CACHE_TTL_SECONDS invalide (valeur reçue: %q)", os.Getenv("CACHE_TTL_SECONDS"))
	}
	c.CacheTTL = time.Duration(ttlSec) * time.Second

	return c, nil
}

// DSN retourne la Data Source Name pour sqlx/MariaDB.
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

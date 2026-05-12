// Package config lit la configuration du bot et du dashboard
// depuis les variables d'environnement.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config regroupe tous les paramètres de l'application.
type Config struct {
	// Discord bot
	DiscordBotToken string

	// Discord OAuth2 (dashboard web)
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURL  string

	// Base de données
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// Serveur web
	WebListenAddr string
	SessionSecret string

	// Logs
	LogLevel string
}

// Load lit et valide la configuration depuis l'environnement.
// Retourne une erreur si une variable obligatoire est absente ou invalide.
func Load() (*Config, error) {
	port, err := envInt("DB_PORT", 3306)
	if err != nil {
		return nil, fmt.Errorf("config: DB_PORT invalide: %w", err)
	}

	cfg := &Config{
		DiscordBotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURL:  os.Getenv("DISCORD_REDIRECT_URL"),
		DBHost:              envDefault("DB_HOST", "mariadb"),
		DBPort:              port,
		DBName:              envDefault("DB_NAME", "discordbot"),
		DBUser:              envDefault("DB_USER", "discordbot"),
		DBPassword:          os.Getenv("DB_PASSWORD"),
		WebListenAddr:       envDefault("WEB_LISTEN_ADDR", ":8080"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		LogLevel:            envDefault("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

// DSN retourne la chaîne de connexion MariaDB pour database/sql.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

// ---

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}

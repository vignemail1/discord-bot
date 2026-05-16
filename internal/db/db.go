// Package db fournit la connexion MariaDB avec retry et pool configuré.
package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 10
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute

	retryAttempts = 10
	retryDelay    = 3 * time.Second
)

// Connect établit la connexion MariaDB avec retry.
// Retourne une erreur si toutes les tentatives échouent.
func Connect(ctx context.Context, dsn string) (*sqlx.DB, error) {
	// sqlx.Open ne valide pas la connexion réseau : on ouvre une fois,
	// puis on tente le ping dans la boucle.
	conn, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	conn.SetMaxOpenConns(maxOpenConns)
	conn.SetMaxIdleConns(maxIdleConns)
	conn.SetConnMaxLifetime(connMaxLifetime)
	conn.SetConnMaxIdleTime(connMaxIdleTime)

	var lastErr error

	for attempt := 1; attempt <= retryAttempts; attempt++ {
		if lastErr = conn.PingContext(ctx); lastErr == nil {
			slog.Info("db: connexion établie", "attempt", attempt)
			return conn, nil
		}

		slog.Warn("db: ping échoué, nouvelle tentative",
			"attempt", attempt,
			"max", retryAttempts,
			"delay", retryDelay,
			"err", lastErr,
		)

		select {
		case <-ctx.Done():
			_ = conn.Close()
			return nil, fmt.Errorf("db: contexte annulé avant connexion: %w", ctx.Err())
		case <-time.After(retryDelay):
		}
	}

	_ = conn.Close()
	return nil, fmt.Errorf("db: impossible de se connecter après %d tentatives: %w", retryAttempts, lastErr)
}

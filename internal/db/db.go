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
	var db *sqlx.DB
	var err error

	for attempt := 1; attempt <= retryAttempts; attempt++ {
		db, err = sqlx.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("db: open: %w", err)
		}

		db.SetMaxOpenConns(maxOpenConns)
		db.SetMaxIdleConns(maxIdleConns)
		db.SetConnMaxLifetime(connMaxLifetime)
		db.SetConnMaxIdleTime(connMaxIdleTime)

		if pingErr := db.PingContext(ctx); pingErr == nil {
			slog.Info("db: connexion établie", "attempt", attempt)
			return db, nil
		}

		slog.Warn("db: ping échoué, nouvelle tentative",
			"attempt", attempt,
			"max", retryAttempts,
			"delay", retryDelay,
		)

		_ = db.Close()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("db: contexte annulé avant connexion: %w", ctx.Err())
		case <-time.After(retryDelay):
		}
	}

	return nil, fmt.Errorf("db: impossible de se connecter après %d tentatives: %w", retryAttempts, err)
}

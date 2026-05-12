// Package db — chargement et application des migrations golang-migrate.
package db

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

// RunMigrations applique toutes les migrations UP non encore appliquées.
// Le dossier migrationsPath doit contenir les fichiers *.up.sql / *.down.sql.
func RunMigrations(db *sqlx.DB, migrationsPath string) error {
	driver, err := mysql.WithInstance(db.DB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("migrations: driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migrations: init: %w", err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("migrations: aucune nouvelle migrationà appliquer")
			return nil
		}
		return fmt.Errorf("migrations: up: %w", err)
	}

	version, dirty, _ := m.Version()
	slog.Info("migrations: appliquées avec succès", "version", version, "dirty", dirty)
	return nil
}

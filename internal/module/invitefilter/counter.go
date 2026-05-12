package invitefilter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// CounterRepository est le contrat de persistance des compteurs d'infractions.
type CounterRepository interface {
	// Increment incrémente le compteur d'un utilisateur et retourne la nouvelle valeur.
	Increment(ctx context.Context, guildID, userID, moduleName string) (int, error)
	// Get retourne le compteur actuel (0 si absent).
	Get(ctx context.Context, guildID, userID, moduleName string) (int, error)
	// Reset remet le compteur à zéro (ex : après un ban).
	Reset(ctx context.Context, guildID, userID, moduleName string) error
}

// MariaDBCounterRepo est l'implémentation MariaDB de CounterRepository.
type MariaDBCounterRepo struct {
	db *sqlx.DB
}

// NewMariaDBCounterRepo crée un nouveau MariaDBCounterRepo.
func NewMariaDBCounterRepo(db *sqlx.DB) *MariaDBCounterRepo {
	return &MariaDBCounterRepo{db: db}
}

func (r *MariaDBCounterRepo) Increment(ctx context.Context, guildID, userID, moduleName string) (int, error) {
	const upsert = `
		INSERT INTO guild_member_module_counters (guild_id, user_id, module_name, count)
		VALUES (?, ?, ?, 1)
		ON DUPLICATE KEY UPDATE count = count + 1
	`
	if _, err := r.db.ExecContext(ctx, upsert, guildID, userID, moduleName); err != nil {
		return 0, fmt.Errorf("counter.Increment: %w", err)
	}
	var count int
	const sel = `SELECT count FROM guild_member_module_counters WHERE guild_id=? AND user_id=? AND module_name=?`
	if err := r.db.GetContext(ctx, &count, sel, guildID, userID, moduleName); err != nil {
		return 0, fmt.Errorf("counter.Increment get: %w", err)
	}
	return count, nil
}

func (r *MariaDBCounterRepo) Get(ctx context.Context, guildID, userID, moduleName string) (int, error) {
	const q = `SELECT count FROM guild_member_module_counters WHERE guild_id=? AND user_id=? AND module_name=?`
	var count int
	if err := r.db.GetContext(ctx, &count, q, guildID, userID, moduleName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("counter.Get: %w", err)
	}
	return count, nil
}

func (r *MariaDBCounterRepo) Reset(ctx context.Context, guildID, userID, moduleName string) error {
	const q = `DELETE FROM guild_member_module_counters WHERE guild_id=? AND user_id=? AND module_name=?`
	if _, err := r.db.ExecContext(ctx, q, guildID, userID, moduleName); err != nil {
		return fmt.Errorf("counter.Reset: %w", err)
	}
	return nil
}

// MemoryCounterRepo est une implémentation en mémoire pour les tests.
type MemoryCounterRepo struct {
	counts map[string]int
	IncrErr error
	GetErr  error
	ResetErr error
}

func NewMemoryCounterRepo() *MemoryCounterRepo {
	return &MemoryCounterRepo{counts: make(map[string]int)}
}

func key(guildID, userID, moduleName string) string {
	return guildID + "|" + userID + "|" + moduleName
}

func (r *MemoryCounterRepo) Increment(ctx context.Context, guildID, userID, moduleName string) (int, error) {
	if r.IncrErr != nil {
		return 0, r.IncrErr
	}
	r.counts[key(guildID, userID, moduleName)]++
	return r.counts[key(guildID, userID, moduleName)], nil
}

func (r *MemoryCounterRepo) Get(ctx context.Context, guildID, userID, moduleName string) (int, error) {
	if r.GetErr != nil {
		return 0, r.GetErr
	}
	return r.counts[key(guildID, userID, moduleName)], nil
}

func (r *MemoryCounterRepo) Reset(ctx context.Context, guildID, userID, moduleName string) error {
	if r.ResetErr != nil {
		return r.ResetErr
	}
	delete(r.counts, key(guildID, userID, moduleName))
	return nil
}

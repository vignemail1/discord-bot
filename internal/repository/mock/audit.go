package mock

import (
	"context"
	"sort"
	"sync"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// AuditRepositoryMock est une implémentation en mémoire de repository.AuditRepository.
type AuditRepositoryMock struct {
	mu     sync.RWMutex
	events []repository.AuditEvent
	ListErr error
}

func NewAuditRepository() *AuditRepositoryMock {
	return &AuditRepositoryMock{}
}

// Add insère un événement dans le mock (helper de test).
func (m *AuditRepositoryMock) Add(e repository.AuditEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *AuditRepositoryMock) ListEvents(ctx context.Context, guildID string, f repository.AuditFilter) ([]repository.AuditEvent, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []repository.AuditEvent
	for _, e := range m.events {
		if e.GuildID != guildID {
			continue
		}
		if f.UserID != "" && e.UserID != f.UserID {
			continue
		}
		if f.EventType != "" && e.EventType != f.EventType {
			continue
		}
		if f.Before > 0 && e.ID >= f.Before {
			continue
		}
		out = append(out, e)
	}

	// Tri décroissant par ID (plus récent en premier).
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

var _ repository.AuditRepository = (*AuditRepositoryMock)(nil)

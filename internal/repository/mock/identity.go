package mock

import (
	"context"
	"sort"
	"sync"

	"github.com/vignemail1/discord-bot/internal/repository"
)

// IdentityRepositoryMock est une implémentation en mémoire de repository.IdentityRepository.
type IdentityRepositoryMock struct {
	mu     sync.RWMutex
	states map[string]repository.IdentityState  // clé: guildID+":"+userID
	events []repository.IdentityEvent

	ListMembersErr    error
	GetMemberErr      error
	ListEventsErr     error
}

func NewIdentityRepository() *IdentityRepositoryMock {
	return &IdentityRepositoryMock{
		states: make(map[string]repository.IdentityState),
	}
}

func stateKey(guildID, userID string) string { return guildID + ":" + userID }

// AddState insère ou remplace un état membre (helper de test).
func (m *IdentityRepositoryMock) AddState(s repository.IdentityState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[stateKey(s.GuildID, s.UserID)] = s
}

// AddEvent insère un événement (helper de test).
func (m *IdentityRepositoryMock) AddEvent(e repository.IdentityEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *IdentityRepositoryMock) ListMembers(ctx context.Context, guildID string) ([]repository.IdentityState, error) {
	if m.ListMembersErr != nil {
		return nil, m.ListMembersErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []repository.IdentityState
	for _, s := range m.states {
		if s.GuildID == guildID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *IdentityRepositoryMock) GetMember(ctx context.Context, guildID, userID string) (*repository.IdentityState, error) {
	if m.GetMemberErr != nil {
		return nil, m.GetMemberErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.states[stateKey(guildID, userID)]
	if !ok {
		return nil, nil
	}
	copy := s
	return &copy, nil
}

func (m *IdentityRepositoryMock) ListMemberEvents(ctx context.Context, guildID, userID string, f repository.IdentityFilter) ([]repository.IdentityEvent, error) {
	if m.ListEventsErr != nil {
		return nil, m.ListEventsErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []repository.IdentityEvent
	for _, e := range m.events {
		if e.GuildID != guildID || e.UserID != userID {
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

var _ repository.IdentityRepository = (*IdentityRepositoryMock)(nil)

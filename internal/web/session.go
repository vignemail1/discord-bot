package web

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// sessionCookieName est le nom du cookie de session HTTP.
	sessionCookieName = "discord_bot_session"
	// sessionTTL est la durée de vie d'une session authentifiée.
	sessionTTL = 24 * time.Hour
	// sessionGCInterval est l'intervalle de nettoyage des sessions expirées.
	sessionGCInterval = 30 * time.Minute
)

// Session représente une session utilisateur dashboard.
type Session struct {
	ID           string
	StateToken   string    // token CSRF pour le flux OAuth2
	UserID       string    // Discord user_id (vide avant auth)
	Username     string    // Discord username
	GlobalName   string    // Discord display name
	AvatarHash   string    // Discord avatar hash
	AccessToken  string    // token OAuth2
	RefreshToken string    // token de rafraîchissement
	TokenExpiry  time.Time // expiration du token OAuth2
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// IsAuthenticated retourne true si la session est associée à un utilisateur.
func (s *Session) IsAuthenticated() bool {
	return s != nil && s.UserID != ""
}

// sessionEntry encapsule une session avec son expiration dans le store.
type sessionEntry struct {
	session   Session
	expiresAt time.Time
}

// SessionStore est un store de sessions en mémoire thread-safe.
// Pas de persistance DB : les sessions sont perdues au redémarrage (comportement voulu).
type SessionStore struct {
	mu      sync.RWMutex
	entries map[string]*sessionEntry
}

// NewSessionStore crée un store vide.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		entries: make(map[string]*sessionEntry),
	}
}

// Create génère une nouvelle session vide (pré-auth) et la stocke.
func (s *SessionStore) Create() (*Session, error) {
	id, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	state, err := generateToken(16)
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID:         id,
		StateToken: state,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(sessionTTL),
	}
	s.mu.Lock()
	s.entries[id] = &sessionEntry{session: *sess, expiresAt: sess.ExpiresAt}
	s.mu.Unlock()
	return sess, nil
}

// Get retourne la session par son ID. Retourne nil si absente ou expirée.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	e, ok := s.entries[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	if time.Now().After(e.expiresAt) {
		s.Delete(id)
		return nil
	}
	copy := e.session
	return &copy
}

// Save persiste les modifications d'une session existante.
func (s *SessionStore) Save(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.entries[sess.ID]; ok {
		e.session = *sess
	}
}

// Delete supprime une session du store.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.entries, id)
	s.mu.Unlock()
}

// GC supprime les sessions expirées. À appeler périodiquement.
func (s *SessionStore) GC() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, id)
		}
	}
}

// generateToken génère un token hexadécimal aléatoire de n octets.
func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

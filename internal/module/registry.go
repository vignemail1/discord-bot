package module

import (
	"fmt"
	"sync"
)

// Registry maintient la liste des modules enregistrés pour l'ensemble du bot.
// Thread-safe : les modules sont enregistrés au démarrage avant que
// la Gateway soit ouverte ; les lectures sont concurrentes.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module
	ordered []string // conserve l'ordre d'enregistrement pour le dispatch
}

// NewRegistry crée un registre vide.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

// Register ajoute un module au registre.
// Retourne une erreur si un module portant le même nom est déjà enregistré.
func (r *Registry) Register(m Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := m.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("registry: module %q déjà enregistré", name)
	}
	r.modules[name] = m
	r.ordered = append(r.ordered, name)
	return nil
}

// MustRegister appelle Register et panique si le nom est dupliqué.
// Réservé à l'initialisation dans main().
func (r *Registry) MustRegister(m Module) {
	if err := r.Register(m); err != nil {
		panic(err)
	}
}

// Get retourne un module par nom.
func (r *Registry) Get(name string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[name]
	return m, ok
}

// All retourne tous les modules dans l'ordre d'enregistrement.
func (r *Registry) All() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Module, 0, len(r.ordered))
	for _, name := range r.ordered {
		out = append(out, r.modules[name])
	}
	return out
}

// Names retourne les noms de tous les modules enregistrés.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.ordered))
	copy(out, r.ordered)
	return out
}

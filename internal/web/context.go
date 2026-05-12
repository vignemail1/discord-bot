// Package web implémente le dashboard HTTP du bot Discord.
package web

import "context"

// contextKey est un type opaque pour les clés de contexte de ce package.
// Évite les collisions avec d'autres packages.
type contextKey int

const (
	contextKeySession contextKey = iota
	contextKeyUserID
)

// sessionFromContext extrait la session du contexte.
func sessionFromContext(ctx context.Context) *Session {
	v, _ := ctx.Value(contextKeySession).(*Session)
	return v
}

// userIDFromContext extrait le user_id Discord du contexte.
func userIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyUserID).(string)
	return v
}

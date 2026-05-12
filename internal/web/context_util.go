package web

import "context"

// contextSet insère une valeur typée dans le contexte.
func contextSet(ctx interface{ Value(any) any }, key contextKey, val any) context.Context {
	switch c := ctx.(type) {
	case context.Context:
		return context.WithValue(c, key, val)
	default:
		panic("contextSet: ctx n'implémente pas context.Context")
	}
}

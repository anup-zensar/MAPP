// Package actor carries who is acting, the signed-in person or the system; it imports nothing from MAPP.
package actor

import "context"

type Actor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var System = Actor{ID: "system", Name: "SP&A One"} //nolint:gochecknoglobals // a fixed, read-only actor used by 75 callers

type contextKey struct{}

func With(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, contextKey{}, a)
}

func From(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(contextKey{}).(Actor)
	return a, ok
}

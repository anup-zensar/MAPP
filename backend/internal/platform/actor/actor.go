// Package actor says who is acting: the signed-in person behind a request.
//
// The sign-in middleware puts the actor in the request context; handlers take it out with From and pass
// it explicitly to every service method, so every change can be recorded against a person (FR-007,
// research R7). The package imports nothing from MAPP, so every other package can use it without
// import cycles.
package actor

import "context"

// Actor is a person (or the system) doing something in MAPP. Records store it as
// {"id": "U-1002", "name": "Priya Raman"}, with the name as it was at the time.
type Actor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// System is the actor for work MAPP does by itself, such as loading the demo data.
var System = Actor{ID: "system", Name: "SP&A One"} //nolint:gochecknoglobals // a fixed, read-only actor used by 75 callers

// contextKey is private, so no other package can overwrite the actor by accident.
type contextKey struct{}

// With returns a copy of ctx that carries a.
func With(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, contextKey{}, a)
}

// From returns the actor carried by ctx, and false when there is none.
func From(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(contextKey{}).(Actor)
	return a, ok
}

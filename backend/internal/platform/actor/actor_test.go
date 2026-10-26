package actor_test

import (
	"context"
	"encoding/json"
	"testing"

	"mapp/internal/platform/actor"
)

// The sign-in middleware puts the actor in the request context and handlers take it out again, so every
// business operation knows who acts (FR-007).
func TestWithAndFrom_CarryTheActor_FR007(t *testing.T) {
	priya := actor.Actor{ID: "U-1002", Name: "Priya Raman"}

	got, ok := actor.From(actor.With(context.Background(), priya))
	if !ok {
		t.Fatal("From found no actor in a context made by With")
	}
	if got != priya {
		t.Errorf("From = %+v, want %+v", got, priya)
	}
}

func TestFrom_NoActorInContext(t *testing.T) {
	if a, ok := actor.From(context.Background()); ok {
		t.Errorf("From = %+v, true; want false for a context without an actor", a)
	}
}

// Records store the actor as {"id", "name"} (data model "Actor reference"); the system actor creates
// the seed records.
func TestActor_JSON(t *testing.T) {
	body, err := json.Marshal(actor.System)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if got, want := string(body), `{"id":"system","name":"MAPP"}`; got != want {
		t.Errorf("JSON = %s, want %s", got, want)
	}
}

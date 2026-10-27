package history

import (
	"testing"
)

// A kind that history could not work with stops the API from starting (research R8): every mistake is
// refused with an error, and a refused kind is not registered.
func TestRegister_RefusesBadKinds(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	parent, child := kinds[0], kinds[1]
	if err := f.service.Register(parent); err != nil {
		t.Fatalf("Register(parent): %v", err)
	}
	mistakes := []struct {
		name   string
		change func(k *Kind) // applied to a copy of the child kind
	}{
		{"no name", func(k *Kind) { k.Name = "" }},
		{"a name that is already registered", func(k *Kind) { k.Name, k.Parent = "parent", "" }},
		{"itself as its parent", func(k *Kind) { k.Parent = k.Name }},
		{"a parent that is not registered yet", func(k *Kind) { k.Parent = "grandparent" }},
		{"no Snapshot", func(k *Kind) { k.Snapshot = nil }},
		{"no Remove", func(k *Kind) { k.Remove = nil }},
		{"no Put", func(k *Kind) { k.Put = nil }},
		{"a parent but no ChildIDs", func(k *Kind) { k.ChildIDs = nil }},
	}
	for _, mistake := range mistakes {
		t.Run(mistake.name, func(t *testing.T) {
			bad := child
			mistake.change(&bad)

			if err := f.service.Register(bad); err == nil {
				t.Errorf("Register accepted a kind with %s", mistake.name)
			}
		})
	}

	if _, found := f.service.findKind("child"); found {
		t.Error("a refused kind was registered")
	}
	if err := f.service.Register(child); err != nil {
		t.Errorf("Register(child) = %v, want nil", err)
	}
}

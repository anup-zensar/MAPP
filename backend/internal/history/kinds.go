package history

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"northstar/internal/platform/actor"
)

// Kind describes one record type to history, so that history can log its changes, move a record of it to
// the recycle bin with everything under it, and bring it back (research R8). Each business package
// registers a Kind for every record type it owns, parents before children, while the API starts.
//
// The functions run inside the caller's transaction, which their context carries.
type Kind struct {
	Name     string // "sampleNote": stored in log and bin entries
	Singular string // "sample note": mid-sentence; capitalise the first letter at the start of a sentence
	Plural   string // "sample notes"
	Parent   string // the parent kind's Name; "" for top-level records

	// Snapshot reads the record with id: its ID, Name, ParentID and Data (the record as stored); history
	// fills in the snapshot's Kind. It returns false when there is no such record.
	Snapshot func(ctx context.Context, id string) (Snapshot, bool, error)
	// ChildIDs lists the ids of the records of this kind under the parent parentID. Required when Parent is
	// set.
	ChildIDs func(ctx context.Context, parentID string) ([]string, error)
	// Remove deletes the record with id from storage.
	Remove func(ctx context.Context, id string) error
	// Put writes the record of a snapshot back to storage, exactly as its Data says.
	Put func(ctx context.Context, s Snapshot) error

	// The hooks below are optional.

	// Describe gives a field's label and a value's text for the activity log, such as "Owner" and "Priya
	// Raman" for a person's id. value is the stored JSON value, decoded: nil, bool, json.Number, string,
	// []any or map[string]any. An empty label or text means "use the default".
	Describe func(ctx context.Context, field string, value any) (label, text string, err error)
	// PrepareRestore checks or adjusts a record just before it is put back, and may refuse the restore with
	// an error (an Undo of a name that is now taken). When it changes the record, it changes both s.Data and
	// s.Name, and records the change in the record's record.Meta with Touch. It returns short notes for the
	// log and the person, such as "Renamed to 'Ideas (restored)' because the name was taken."
	PrepareRestore func(ctx context.Context, rc RestoreContext, s *Snapshot) (notes []string, err error)
}

// RestoreContext tells PrepareRestore who is restoring, when, and whether it is an Undo, which must put the
// record back exactly, or a Restore from the recycle bin, which applies the kind's restore rules.
type RestoreContext struct {
	Actor actor.Actor
	Now   time.Time
	Undo  bool
}

// Register adds k to the kinds history knows. It is called while the API starts, before the API serves
// requests, and parents must be registered before their children (which also rules out cycles). It refuses
// a kind without a name or with a name already registered, a kind that is its own parent or whose parent is
// not registered yet, and a kind without Snapshot, Remove or Put, or without ChildIDs when it has a parent.
// A refusal stops the API from starting.
func (s *Service) Register(k Kind) error {
	_, alreadyRegistered := s.findKind(k.Name)
	_, parentKnown := s.findKind(k.Parent)
	switch {
	case k.Name == "":
		return errors.New("history: a kind needs a name")
	case alreadyRegistered:
		return fmt.Errorf("history: the kind %q is already registered", k.Name)
	case k.Parent == k.Name:
		return fmt.Errorf("history: the kind %q cannot be its own parent", k.Name)
	case k.Parent != "" && !parentKnown:
		return fmt.Errorf("history: the parent %q of the kind %q is not registered yet; register parents first", k.Parent, k.Name)
	case k.Snapshot == nil || k.Remove == nil || k.Put == nil:
		return fmt.Errorf("history: the kind %q needs the functions Snapshot, Remove and Put", k.Name)
	case k.Parent != "" && k.ChildIDs == nil:
		return fmt.Errorf("history: the kind %q has a parent, so it needs the function ChildIDs", k.Name)
	}
	s.kinds = append(s.kinds, k)
	return nil
}

// findKind returns the registered kind called name, and false when there is none.
func (s *Service) findKind(name string) (Kind, bool) {
	i := slices.IndexFunc(s.kinds, func(k Kind) bool { return k.Name == name })
	if i < 0 {
		return Kind{}, false
	}
	return s.kinds[i], true
}

// registeredKind returns the kind called name. A name that was never registered is a mistake in the calling
// package, so it is an error for the log, not a message for the person.
func (s *Service) registeredKind(name string) (Kind, error) {
	k, found := s.findKind(name)
	if !found {
		return Kind{}, fmt.Errorf("history: the kind %q is not registered", name)
	}
	return k, nil
}

// childKinds returns the kinds whose parent is the kind called name, in the order they were registered.
func (s *Service) childKinds(name string) []Kind {
	var children []Kind
	for _, k := range s.kinds {
		if k.Parent == name {
			children = append(children, k)
		}
	}
	return children
}

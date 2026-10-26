package history

import (
	"context"
	"errors"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// Access is what history asks of access control (feature 014, D-238): which log entries and recycle-bin
// entries the person acting may see, and whether they may bring a deleted record back. internal/app
// implements it (history_access.go). Every method runs inside the caller's transaction and reads the person
// from ctx; a context without a person (an internal call) may see and restore everything.
type Access interface {
	// CanSeeEntry reports whether the person acting may see the log entry e.
	CanSeeEntry(ctx context.Context, e Entry) bool
	// CanSeeBinEntry reports whether the person acting may see the recycle-bin entry e.
	CanSeeBinEntry(ctx context.Context, e BinEntry) bool
	// CanRestore refuses, with 403 forbidden, an Undo or Restore of e that the person acting may not make.
	CanRestore(ctx context.Context, e BinEntry) error
}

// UseAccess tells history about access control (feature 014). Until it is called, everyone sees the whole
// log and the whole recycle bin and may restore everything.
func (s *Service) UseAccess(a Access) {
	s.access = a
}

// canSeeEntry reports whether the person acting may see the log entry e.
func (s *Service) canSeeEntry(ctx context.Context, e Entry) bool {
	return s.access == nil || s.access.CanSeeEntry(ctx, e)
}

// canSeeBinEntry reports whether the person acting may see the recycle-bin entry e.
func (s *Service) canSeeBinEntry(ctx context.Context, e BinEntry) bool {
	return s.access == nil || s.access.CanSeeBinEntry(ctx, e)
}

// canRestore refuses an Undo or Restore of e that the person acting may not make.
func (s *Service) canRestore(ctx context.Context, e BinEntry) error {
	if s.access == nil {
		return nil
	}
	return s.access.CanRestore(ctx, e)
}

// canRestoreAll refuses the restore of entries, all of which come back together, when the person acting
// may not restore one of them.
func (s *Service) canRestoreAll(ctx context.Context, entries []BinEntry) error {
	for _, e := range entries {
		if err := s.canRestore(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// visibleEntries returns the entries of the log the person acting may see, in the same order.
func (s *Service) visibleEntries(ctx context.Context, entries []Entry) []Entry {
	if s.access == nil {
		return entries
	}
	visible := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if s.access.CanSeeEntry(ctx, e) {
			visible = append(visible, e)
		}
	}
	return visible
}

// binEntryAccess returns what the person acting may do with the recycle-bin entry e (BinEntryAccess,
// feature 014 FR-011): restore it (also Undo and the restore preview) when CanRestore allows it, with its
// refusal as the reason otherwise, and delete it permanently with "Delete permanently from the recycle
// bin". Nil for an internal call.
func (s *Service) binEntryAccess(ctx context.Context, e BinEntry) (*authz.BinEntryAccess, error) {
	if _, restricted := authz.From(ctx); !restricted {
		return nil, nil
	}
	restore := authz.Allowed
	if err := s.canRestore(ctx, e); err != nil {
		var refusal *errs.Error
		if !errors.As(err, &refusal) {
			return nil, err
		}
		restore = authz.Unavailable(refusal.Message)
	}
	return &authz.BinEntryAccess{Restore: restore, DeletePermanently: authz.AvailableAnywhere(ctx, authz.RecycleBinPurge)}, nil
}

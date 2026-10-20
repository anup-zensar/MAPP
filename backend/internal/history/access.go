package history

import "context"

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

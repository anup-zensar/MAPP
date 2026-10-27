package history

import "context"

// SnapshotsInBin returns the snapshots of the kind called kindName that are in the recycle bin: deleted
// records and the records under them, from the entries not yet undone or restored, oldest entry first and
// in each entry's order (parents before children). It joins the caller's transaction. Other features use
// it to count records in the bin as still in use (feature 002, data model §10). Never nil.
func (s *Service) SnapshotsInBin(ctx context.Context, kindName string) ([]Snapshot, error) {
	snapshots := []Snapshot{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		entries, err := s.repo.BinEntries(ctx) // oldest first
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Resolution != nil {
				continue
			}
			for _, snapshot := range e.Snapshots {
				if snapshot.Kind == kindName {
					snapshots = append(snapshots, snapshot)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

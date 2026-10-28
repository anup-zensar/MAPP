package history

import "context"

// DeleteGuard may refuse a delete once MoveToBin has collected what it removes; then nothing is removed or logged.
type DeleteGuard func(ctx context.Context, snapshots []Snapshot) error

func (s *Service) UseDeleteGuard(g DeleteGuard) {
	s.guards = append(s.guards, g)
}

func (s *Service) guarded(ctx context.Context, snapshots []Snapshot) error {
	for _, guard := range s.guards {
		if err := guard(ctx, snapshots); err != nil {
			return err
		}
	}
	return nil
}

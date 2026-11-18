package applications

import (
	"context"
	"errors"
)

// ErrSessionUnavailable is returned when a tenant session id does not exist on schema 5.
var ErrSessionUnavailable = errSessionUnavailable

func (s *Store) mysqlSessionSchema5(ctx context.Context) (bool, error) {
	if s.dialect != "mysql" || s.db == nil {
		return false, nil
	}
	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return false, err
	}
	return version >= 5 && version <= 8, nil
}

// TryMutateSessionsV5 applies session mutations on schema 5 relational storage.
func (s *Store) TryMutateSessionsV5(ctx context.Context, fn func(*State) error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema5, err := s.mysqlSessionSchema5(ctx)
	if err != nil {
		return false, err
	}
	if !schema5 {
		return false, nil
	}
	if err = mutateMySQLSessionsV5(ctx, s.db, fn); err != nil {
		if errors.Is(err, errSessionUnavailable) {
			return true, err
		}
		return true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return true, err
	}
	return true, nil
}
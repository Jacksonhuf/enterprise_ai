package applications

import (
	"context"
	"errors"
)

// ErrGrantUnavailable is returned when a tenant grant id does not exist on schema 4.
var ErrGrantUnavailable = errGrantUnavailable

func (s *Store) mysqlGrantSchema4(ctx context.Context) (bool, error) {
	if s.dialect != "mysql" || s.db == nil {
		return false, nil
	}
	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return false, err
	}
	return version == 4, nil
}

// TryMutateGrantsV4 applies grant create/revoke mutations on schema 4 relational storage.
func (s *Store) TryMutateGrantsV4(ctx context.Context, fn func(*State) error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema4, err := s.mysqlGrantSchema4(ctx)
	if err != nil {
		return false, err
	}
	if !schema4 {
		return false, nil
	}
	if err = mutateMySQLGrantsV4(ctx, s.db, fn); err != nil {
		if errors.Is(err, errGrantUnavailable) {
			return true, err
		}
		return true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return true, err
	}
	return true, nil
}
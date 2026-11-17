package applications

import (
	"context"
	"errors"
	"time"
)

// Exported for HTTP error mapping at the API boundary.
var (
	ErrRelationalAppUnavailable    = errRelationalAppUnavailable
	ErrRelationalDraftConflict     = errRelationalDraftConflict
	ErrRelationalCheckInvalid      = errRelationalCheckInvalid
	ErrRelationalReleaseConflict   = errRelationalReleaseConflict
	ErrRelationalApplicationDomain = errRelationalApplicationDomain
)

func (s *Store) mysqlApplicationSchema3(ctx context.Context) (bool, error) {
	if s.dialect != "mysql" || s.db == nil {
		return false, nil
	}
	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return false, err
	}
	return mySQLRelationalRuntimeVersion(version), nil
}

func (s *Store) syncOperationalState(ctx context.Context) error {
	st, err := s.readSQLContextAudit(ctx, false)
	if err != nil {
		return err
	}
	s.state = st
	return nil
}

func (s *Store) TryCreateApplicationV3(ctx context.Context, tenant, owner string, cfg Config, template string, expected *string) (App, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema3, err := s.mysqlApplicationSchema3(ctx)
	if err != nil {
		return App{}, false, err
	}
	if !schema3 {
		return App{}, false, nil
	}
	a, err := createMySQLV3Application(ctx, s.db, tenant, owner, cfg, template, expected)
	if err != nil {
		return App{}, true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return App{}, true, err
	}
	return a, true, nil
}

func (s *Store) TrySaveApplicationDraftV3(ctx context.Context, tenant, id, actor, client, request, expected string, next Config) (App, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema3, err := s.mysqlApplicationSchema3(ctx)
	if err != nil {
		return App{}, false, err
	}
	if !schema3 {
		return App{}, false, nil
	}
	revision, err := saveMySQLV3Draft(ctx, s.db, tenant, id, actor, client, request, expected, next)
	if err != nil {
		return App{}, true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return App{}, true, err
	}
	a, ok := s.state.Apps[id]
	if !ok || a.Revision != revision {
		return App{}, true, errors.New("relational draft save reconciliation failed")
	}
	return a, true, nil
}

func (s *Store) TryRecordReleaseCheckV3(ctx context.Context, tenant, id, actor, expected, validation string, expires time.Time) (Check, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema3, err := s.mysqlApplicationSchema3(ctx)
	if err != nil {
		return Check{}, false, err
	}
	if !schema3 {
		return Check{}, false, nil
	}
	c, err := recordMySQLV3Check(ctx, s.db, tenant, id, actor, expected, validation, expires)
	if err != nil {
		return Check{}, true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return Check{}, true, err
	}
	return c, true, nil
}

func (s *Store) TryPublishReleaseV3(ctx context.Context, tenant, id, actor, client, request, check, expected string, current *string, note string) (Release, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema3, err := s.mysqlApplicationSchema3(ctx)
	if err != nil {
		return Release{}, false, err
	}
	if !schema3 {
		return Release{}, false, nil
	}
	r, err := publishMySQLV3Release(ctx, s.db, tenant, id, actor, client, request, check, expected, current, note)
	if err != nil {
		return Release{}, true, err
	}
	if err = s.syncOperationalState(ctx); err != nil {
		return Release{}, true, err
	}
	return r, true, nil
}

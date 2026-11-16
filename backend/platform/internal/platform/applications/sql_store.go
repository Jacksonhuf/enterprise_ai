package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) readSQL() (State, error) { return s.readSQLContext(context.Background()) }
func (s *Store) readSQLContext(parent context.Context) (State, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	var raw []byte
	var version int
	if e := s.db.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); e != nil {
		return State{}, e
	}
	if version != 1 {
		return State{}, fmt.Errorf("unsupported state schema")
	}
	var st State
	e := json.Unmarshal(raw, &st)
	if e == nil {
		initializeState(&st)
	}
	return st, e
}
func (s *Store) updateSQLContext(parent context.Context, fn func(*State) error) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var raw []byte
	var version int
	selectSQL := `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`
	if s.dialect == "postgres" || s.dialect == "mysql" {
		selectSQL += " FOR UPDATE"
	}
	if e = tx.QueryRowContext(ctx, selectSQL).Scan(&version, &raw); e != nil {
		return e
	}
	if version != 1 {
		return fmt.Errorf("unsupported state schema")
	}
	var next State
	if e = json.Unmarshal(raw, &next); e != nil {
		return e
	}
	initializeState(&next)
	if e = fn(&next); e != nil {
		return e
	}
	b, e := json.Marshal(next)
	if e != nil {
		return e
	}
	updateSQL := `UPDATE cognara_fg_state SET body=? WHERE id=1`
	if s.dialect == "postgres" {
		updateSQL = `UPDATE cognara_fg_state SET body=$1::jsonb WHERE id=1`
	}
	if _, e = tx.ExecContext(ctx, updateSQL, string(b)); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.state = next
	return nil
}
func (s *Store) Check() error {
	if s.db == nil {
		return nil
	}
	_, err := s.readSQL()
	return err
}
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

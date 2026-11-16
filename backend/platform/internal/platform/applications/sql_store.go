package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) readSQL() (State, error) { return s.readSQLContext(context.Background()) }
func (s *Store) readSQLContext(parent context.Context) (State, error) {
	return s.readSQLContextAudit(parent, true)
}
func (s *Store) readSQLContextAudit(parent context.Context, includeAudit bool) (State, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if s.dialect == "mysql" {
		tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
		if e != nil {
			return State{}, e
		}
		defer tx.Rollback()
		st, _, e := readStateRow(ctx, tx, false, true, includeAudit)
		if e != nil {
			return State{}, e
		}
		return st, tx.Commit()
	}
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
func readStateRow(ctx context.Context, tx *sql.Tx, lock, relational, includeAudit bool) (State, int, error) {
	st, version, _, e := readStateRowSource(ctx, tx, lock, relational, includeAudit)
	return st, version, e
}
func readStateRowSource(ctx context.Context, tx *sql.Tx, lock, relational, includeAudit bool) (State, int, []byte, error) {
	query := `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`
	if lock {
		query += " FOR UPDATE"
	}
	var raw []byte
	var version int
	var st State
	if e := tx.QueryRowContext(ctx, query).Scan(&version, &raw); e != nil {
		return st, version, raw, e
	}
	if version != 1 && !(relational && version == 2) {
		return st, version, raw, fmt.Errorf("unsupported state schema")
	}
	if e := json.Unmarshal(raw, &st); e != nil {
		return st, version, raw, e
	}
	if version == 2 {
		if e := validateMySQLAuditSchema(ctx, tx); e != nil {
			return st, version, raw, e
		}
		if len(st.Audit) != 0 {
			return st, version, raw, fmt.Errorf("ambiguous audit storage")
		}
		var e error
		if includeAudit {
			st.Audit, e = loadMySQLAudit(ctx, tx)
		}
		if e != nil {
			return st, version, raw, e
		}
	}
	initializeState(&st)
	return st, version, raw, nil
}

func (s *Store) updateSQLContext(parent context.Context, fn func(*State) error, operational bool) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	next, version, source, e := readStateRowSource(ctx, tx, s.dialect == "postgres" || s.dialect == "mysql", s.dialect == "mysql", !operational)
	if e != nil {
		return e
	}
	previousAudit := append([]Event(nil), next.Audit...)
	if e = fn(&next); e != nil {
		return e
	}
	stored := next
	if version == 2 {
		if operational {
			if len(next.Audit) > 0 {
				var last uint64
				if e = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); e != nil {
					return e
				}
				if e = insertMySQLAudit(ctx, tx, next.Audit, last); e != nil {
					return e
				}
			}
		} else if e = appendMySQLAudit(ctx, tx, previousAudit, next.Audit); e != nil {
			return e
		}
		stored.Audit = nil
	}
	b, e := json.Marshal(stored)
	if e != nil {
		return e
	}
	if s.dialect == "mysql" {
		b, e = preserveMySQLStateEnvelope(source, b)
		if e != nil {
			return e
		}
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
	if operational && version == 2 {
		next.Audit = nil
	}
	s.state = next
	return nil
}
func (s *Store) Check() error {
	if s.dialect == "mysql" {
		_, _, _, e := s.QueryMySQLAudit(context.Background(), "", "", 1, func(Event) bool { return false })
		return e
	}
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

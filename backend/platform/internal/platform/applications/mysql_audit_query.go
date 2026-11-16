package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// QueryMySQLAudit uses tenant/sequence indexes and bounded batches. A predicate keeps
// the existing Go search/time semantics, including legacy tenant mapping and NULL dates.
// Empty tenant is a readiness probe: inspect state/schema without scanning audit rows.
func (s *Store) QueryMySQLAudit(parent context.Context, tenant, after string, limit int, match func(Event) bool) ([]Event, bool, bool, error) {
	if s.dialect != "mysql" {
		return nil, false, false, nil
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return nil, false, false, e
	}
	defer tx.Rollback()
	_, version, e := readStateRow(ctx, tx, false, true, false)
	if e != nil {
		return nil, false, false, e
	}
	if version != 2 || tenant == "" {
		return nil, false, version == 2, tx.Commit()
	}
	if limit < 1 || limit > 100 || match == nil {
		return nil, false, true, errors.New("invalid audit query")
	}
	decode := func(raw []byte, id, tenantID string) (Event, error) {
		var event Event
		if json.Unmarshal(raw, &event) != nil || event.ID != id || event.TenantID != tenantID {
			return event, errors.New("invalid relational audit record")
		}
		return event, nil
	}
	var before uint64
	if after != "" {
		rows, e := tx.QueryContext(ctx, `SELECT sequence_no,event_id,tenant_id,payload FROM cognara_fg_audit WHERE tenant_id IN(?, '') AND event_id=? ORDER BY sequence_no DESC`, tenant, after)
		if e != nil {
			return nil, false, true, e
		}
		for rows.Next() {
			var seq uint64
			var id, t string
			var raw []byte
			if e = rows.Scan(&seq, &id, &t, &raw); e != nil {
				rows.Close()
				return nil, false, true, e
			}
			if seq == 0 {
				rows.Close()
				return nil, false, true, errors.New("invalid audit sequence")
			}
			event, err := decode(raw, id, t)
			if err != nil {
				rows.Close()
				return nil, false, true, err
			}
			if before == 0 && match(event) {
				before = seq
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, false, true, e
		}
		if before == 0 {
			return nil, false, true, ErrAuditCursor
		}
	}
	items := []Event{}
	for {
		query := `SELECT sequence_no,event_id,tenant_id,payload FROM cognara_fg_audit WHERE tenant_id IN(?, '')`
		args := []any{tenant}
		if before != 0 {
			query += ` AND sequence_no<?`
			args = append(args, before)
		}
		query += ` ORDER BY sequence_no DESC LIMIT 200`
		rows, e := tx.QueryContext(ctx, query, args...)
		if e != nil {
			return nil, false, true, e
		}
		n := 0
		for rows.Next() {
			var seq uint64
			var id, t string
			var raw []byte
			if e = rows.Scan(&seq, &id, &t, &raw); e != nil {
				rows.Close()
				return nil, false, true, e
			}
			if seq == 0 {
				rows.Close()
				return nil, false, true, errors.New("invalid audit sequence")
			}
			event, err := decode(raw, id, t)
			if err != nil {
				rows.Close()
				return nil, false, true, err
			}
			before = seq
			n++
			if match(event) {
				items = append(items, event)
			}
			if len(items) > limit {
				break
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, false, true, e
		}
		if len(items) > limit {
			return items[:limit], true, true, tx.Commit()
		}
		if n < 200 {
			return items, false, true, tx.Commit()
		}
	}
}

var ErrAuditCursor = errors.New("audit cursor unavailable")

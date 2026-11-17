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
type AuditSQLFilter struct {
	Outcome  string
	From, To *time.Time
}

func (s *Store) QueryMySQLAudit(parent context.Context, tenant, after string, limit int, match func(Event) bool) ([]Event, bool, bool, error) {
	return s.QueryMySQLAuditFiltered(parent, tenant, after, limit, AuditSQLFilter{}, match)
}

// SQL filters are conservative candidate restrictions. The callback remains the
// final authority for exact time/search/legacy-tenant semantics.
func (s *Store) QueryMySQLAuditFiltered(parent context.Context, tenant, after string, limit int, filter AuditSQLFilter, match func(Event) bool) ([]Event, bool, bool, error) {
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
	var version int
	e = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version)
	if e == nil {
		if version == 3 {
			_, _, e = readMySQLApplicationV3(ctx, tx, false)
		} else {
			_, version, e = readStateRow(ctx, tx, false, true, false)
		}
	}
	if e != nil {
		return nil, false, false, e
	}
	if (version != 2 && version != 3) || tenant == "" {
		return nil, false, version == 2 || version == 3, tx.Commit()
	}
	if limit < 1 || limit > 100 || match == nil {
		return nil, false, true, errors.New("invalid audit query")
	}
	if filter.Outcome != "" && !map[string]bool{"success": true, "accepted": true, "denied": true, "failed": true, "partial": true, "canceled": true}[filter.Outcome] {
		return nil, false, true, errors.New("invalid audit outcome")
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, false, true, errors.New("invalid audit time range")
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
		if filter.Outcome != "" {
			query += ` AND (CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.outcome')) AS BINARY)=CAST(? AS BINARY) OR ` + auditNonCanonicalKeys + `)`
			args = append(args, filter.Outcome)
		}
		for _, bound := range []struct {
			stamp *time.Time
			op    string
		}{{filter.From, ">="}, {filter.To, "<="}} {
			if bound.stamp == nil {
				continue
			}
			utc := bound.stamp.UTC()
			if utc.Year() < 1000 || utc.Year() > 9999 {
				continue
			}
			// DATETIME(6) truncates nanoseconds. Include its entire microsecond bucket;
			// NULL projections (valid ancient/out-of-range dates) are post-filtered in Go.
			query += ` AND (occurred_at IS NULL OR ` + auditNonCanonicalKeys + ` OR occurred_at ` + bound.op + ` ?)`
			args = append(args, utc.Format("2006-01-02 15:04:05.000000"))
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

// Go decodes JSON field names case-insensitively, while JSON paths are exact.
// Imported payloads preserve their original keys, including conflicting aliases.
// Any non-lowercase key conservatively bypasses SQL projection filters so the
// existing decoder/predicate decides the result without canonicalizing user data.
const auditNonCanonicalKeys = `CAST(JSON_KEYS(payload) AS BINARY)<>CAST(LOWER(JSON_KEYS(payload)) AS BINARY)`

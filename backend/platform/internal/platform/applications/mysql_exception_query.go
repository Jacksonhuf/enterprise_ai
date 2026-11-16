package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrExceptionPageCursor = errors.New("exception cursor unavailable")

// QueryMySQLExceptions projects tenant-scoped rows in bounded batches. Array
// traversal still happens in MySQL; the callback retains the existing Go filters
// and stable-ID cursor semantics, including rechecking the anchor on each request.
func (s *Store) QueryMySQLExceptions(parent context.Context, tenant, after string, limit int, match func(map[string]any) bool, idOf func(map[string]any) string) ([]json.RawMessage, bool, bool, error) {
	if s.dialect != "mysql" {
		return nil, false, false, nil
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return nil, false, true, e
	}
	defer tx.Rollback()
	var version int
	var keys, audit, kind []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,'$.audit'),JSON_TYPE(JSON_EXTRACT(body,'$.exceptions')) FROM cognara_fg_state WHERE id=1`).Scan(&version, &keys, &audit, &kind); e != nil {
		return nil, false, true, e
	}
	if version == 1 {
		return nil, false, false, nil
	}
	if version != 2 {
		return nil, false, true, errors.New("unsupported state schema")
	}
	if tenant == "" || limit < 1 || limit > 100 || match == nil || idOf == nil {
		return nil, false, true, errors.New("invalid exception query")
	}
	var names []string
	if json.Unmarshal(keys, &names) != nil {
		return nil, false, true, errors.New("invalid state envelope")
	}
	for _, name := range names {
		for _, canonical := range []string{"audit", "exceptions"} {
			if name != canonical && strings.EqualFold(name, canonical) {
				return nil, false, false, nil
			}
		}
	}
	var embedded []json.RawMessage
	if len(audit) > 0 && (json.Unmarshal(audit, &embedded) != nil || len(embedded) > 0) {
		return nil, false, true, errors.New("ambiguous audit storage")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return nil, false, true, e
	}
	if len(kind) > 0 && string(kind) != "NULL" && string(kind) != "ARRAY" {
		return nil, false, true, errors.New("invalid exception storage")
	}
	items := []json.RawMessage{}
	found := after == ""
	var before uint64
	if string(kind) != "ARRAY" {
		if !found {
			return nil, false, true, ErrExceptionPageCursor
		}
		return items, false, true, tx.Commit()
	}
	for {
		query := `SELECT j.pos,JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,']')) FROM cognara_fg_state s CROSS JOIN JSON_TABLE(s.body,'$.exceptions[*]' COLUMNS(pos FOR ORDINALITY)) j WHERE s.id=1 AND JSON_TYPE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].tenant_id')))='STRING' AND CAST(JSON_UNQUOTE(JSON_EXTRACT(s.body,CONCAT('$.exceptions[',j.pos-1,'].tenant_id'))) AS BINARY)=CAST(? AS BINARY)`
		args := []any{tenant}
		if before != 0 {
			query += ` AND j.pos<?`
			args = append(args, before)
		}
		query += ` ORDER BY j.pos DESC LIMIT 200`
		rows, e := tx.QueryContext(ctx, query, args...)
		if e != nil {
			return nil, false, true, e
		}
		n := 0
		for rows.Next() {
			var pos uint64
			var raw []byte
			if e = rows.Scan(&pos, &raw); e != nil {
				rows.Close()
				return nil, false, true, e
			}
			if pos == 0 {
				rows.Close()
				return nil, false, true, errors.New("invalid exception ordinal")
			}
			before = pos
			n++
			var item map[string]any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if decoder.Decode(&item) != nil || item == nil || item["tenant_id"] != tenant {
				rows.Close()
				return nil, false, true, errors.New("invalid exception record")
			}
			if !match(item) {
				continue
			}
			if !found {
				if idOf(item) == after {
					found = true
				}
				continue
			}
			items = append(items, json.RawMessage(raw))
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
			if !found {
				return nil, false, true, ErrExceptionPageCursor
			}
			return items, false, true, tx.Commit()
		}
	}
}

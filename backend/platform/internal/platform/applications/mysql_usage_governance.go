package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func queryMySQLUsageRelational(ctx context.Context, tx *sql.Tx, tenant, after string, limit int, match func(map[string]any) bool, idOf func(map[string]any) string) ([]json.RawMessage, bool, bool, error) {
	if err := validateMySQLUsageSchema(ctx, tx); err != nil {
		return nil, false, true, err
	}
	items := []json.RawMessage{}
	found := after == ""
	var before uint64
	for {
		query := `SELECT ordinal,raw_payload FROM cognara_fg_usage WHERE tenant_id=?`
		args := []any{tenant}
		if before != 0 {
			query += ` AND ordinal<?`
			args = append(args, before)
		}
		query += ` ORDER BY ordinal DESC LIMIT 200`
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, false, true, err
		}
		n := 0
		for rows.Next() {
			var ordinal uint64
			var raw []byte
			if err = rows.Scan(&ordinal, &raw); err != nil {
				rows.Close()
				return nil, false, true, err
			}
			if ordinal == 0 {
				rows.Close()
				return nil, false, true, errors.New("invalid usage ordinal")
			}
			before = ordinal
			n++
			var item map[string]any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if decoder.Decode(&item) != nil || item == nil || item["tenant_id"] != tenant {
				rows.Close()
				return nil, false, true, errors.New("invalid usage record")
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
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, false, true, err
		}
		if len(items) > limit {
			return items[:limit], true, true, nil
		}
		if n < 200 {
			if !found {
				return nil, false, true, ErrGovernancePageCursor
			}
			return items, false, true, nil
		}
	}
}
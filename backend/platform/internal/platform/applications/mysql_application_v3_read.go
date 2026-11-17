package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Preparatory schema3 reader. Only isolated tests call this until writers,
// snapshots and every legacy projection are ready. OpenMySQL still rejects v3.
// Caller must provide a consistent RepeatableRead-or-stronger transaction.
// Return the raw envelope as well as State so future writers cannot mistake a
// typed projection for a lossless representation of extension fields.
func readMySQLApplicationV3(ctx context.Context, tx *sql.Tx, includeAudit bool) (State, []byte, error) {
	var empty State
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &root); err != nil {
		return empty, nil, err
	}
	if version != 3 {
		return empty, nil, errors.New("application relational reader requires schema 3")
	}
	raw, err := loadMySQLApplicationLayoutRoot(ctx, tx, time.Now().UTC(), root)
	if err != nil {
		return empty, nil, err
	}
	var st State
	if err = json.Unmarshal(raw, &st); err != nil {
		return empty, nil, err
	}
	if st.Apps == nil || st.Sessions == nil || len(st.Audit) != 0 {
		return empty, nil, errors.New("invalid relational state or ambiguous audit")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return empty, nil, err
	}
	if includeAudit {
		st.Audit, err = loadMySQLAudit(ctx, tx)
		if err != nil {
			return empty, nil, err
		}
	}
	initializeState(&st)
	return st, raw, nil
}

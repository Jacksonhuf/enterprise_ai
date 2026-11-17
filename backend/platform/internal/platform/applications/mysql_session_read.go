package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// readMySQLApplicationV5 reconstructs schema5: applications and grants from
// relational tables, sessions from session tables, other domains from state.body.
func readMySQLApplicationV5(ctx context.Context, tx *sql.Tx, includeAudit bool) (State, []byte, error) {
	var empty State
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &root); err != nil {
		return empty, nil, err
	}
	if version != 5 {
		return empty, nil, errors.New("session relational reader requires schema 5")
	}
	raw, err := loadMySQLApplicationLayoutRoot(ctx, tx, time.Now().UTC(), root)
	if err != nil {
		return empty, nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return empty, nil, err
	}
	if _, ok := object["grants"]; ok {
		return empty, nil, errors.New("ambiguous grant storage")
	}
	if _, ok := object["sessions"]; ok {
		return empty, nil, errors.New("ambiguous session storage")
	}
	var st State
	if err = json.Unmarshal(raw, &st); err != nil {
		return empty, nil, err
	}
	if st.Apps == nil || len(st.Audit) != 0 {
		return empty, nil, errors.New("invalid relational state or ambiguous audit")
	}
	st.Grants, err = loadMySQLGrantsFromTable(ctx, tx)
	if err != nil {
		return empty, nil, err
	}
	st.Sessions, err = loadMySQLSessionsFromTable(ctx, tx)
	if err != nil {
		return empty, nil, err
	}
	if err = validateMySQLSessionSchema(ctx, tx); err != nil {
		return empty, nil, err
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
	rebuilt, err := json.Marshal(st)
	if err != nil {
		return empty, nil, err
	}
	return st, rebuilt, nil
}
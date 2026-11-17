package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// readMySQLApplicationV7 reconstructs schema7: relational apps, grants, sessions, settings, exceptions;
// other domains from state.body.
func readMySQLApplicationV7(ctx context.Context, tx *sql.Tx, includeAudit bool) (State, []byte, error) {
	var empty State
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &root); err != nil {
		return empty, nil, err
	}
	if version != 7 {
		return empty, nil, errors.New("exception relational reader requires schema 7")
	}
	raw, err := loadMySQLApplicationLayoutRoot(ctx, tx, time.Now().UTC(), root)
	if err != nil {
		return empty, nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return empty, nil, err
	}
	for _, key := range []string{"grants", "sessions", "settings", "setting_revisions", "exceptions"} {
		if _, ok := object[key]; ok {
			return empty, nil, errors.New("ambiguous aggregate storage")
		}
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
	st.Settings, st.SettingRevisions, err = loadMySQLSettingsFromTable(ctx, tx)
	if err != nil {
		return empty, nil, err
	}
	st.Exceptions, err = loadMySQLExceptionsFromTable(ctx, tx)
	if err != nil {
		return empty, nil, err
	}
	if err = validateMySQLExceptionSchema(ctx, tx); err != nil {
		return empty, nil, err
	}
	if err = validateMySQLSettingsSchema(ctx, tx); err != nil {
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
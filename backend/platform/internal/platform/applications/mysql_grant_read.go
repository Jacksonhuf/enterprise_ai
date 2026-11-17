package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func mySQLRelationalRuntimeVersion(version int) bool { return version == 3 || version == 4 }

func readMySQLApplicationRelational(ctx context.Context, tx *sql.Tx, includeAudit bool, version int) (State, []byte, error) {
	switch version {
	case 3:
		return readMySQLApplicationV3(ctx, tx, includeAudit)
	case 4:
		return readMySQLApplicationV4(ctx, tx, includeAudit)
	default:
		var empty State
		return empty, nil, errors.New("relational runtime reader requires schema 3 or 4")
	}
}

// readMySQLApplicationV4 reconstructs schema4: applications from relational tables,
// grants from cognara_fg_grants, other domains from state.body.
func readMySQLApplicationV4(ctx context.Context, tx *sql.Tx, includeAudit bool) (State, []byte, error) {
	var empty State
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &root); err != nil {
		return empty, nil, err
	}
	if version != 4 {
		return empty, nil, errors.New("grant relational reader requires schema 4")
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
	var st State
	if err = json.Unmarshal(raw, &st); err != nil {
		return empty, nil, err
	}
	if st.Apps == nil || st.Sessions == nil || len(st.Audit) != 0 {
		return empty, nil, errors.New("invalid relational state or ambiguous audit")
	}
	st.Grants, err = loadMySQLGrantsFromTable(ctx, tx)
	if err != nil {
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
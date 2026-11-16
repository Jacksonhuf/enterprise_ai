package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

const snapshotV2Format = "cognara-fg-snapshot"

type snapshotV2 struct {
	Format  string            `json:"format"`
	Version int               `json:"schema_version"`
	State   json.RawMessage   `json:"state"`
	Audit   []json.RawMessage `json:"audit"`
}

func exportMySQLV2(ctx context.Context, tx *sql.Tx, state []byte) ([]byte, int, error) {
	if e := validateMySQLAuditSchema(ctx, tx); e != nil {
		return nil, 0, e
	}
	var st State
	if json.Unmarshal(state, &st) != nil || st.Apps == nil || st.Sessions == nil || len(st.Audit) != 0 {
		return nil, 0, errors.New("ambiguous snapshot audit")
	}
	rows, e := tx.QueryContext(ctx, `SELECT tenant_id,event_id,sequence_no,payload FROM cognara_fg_audit ORDER BY sequence_no`)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	audit := []json.RawMessage{}
	size := len(state)
	for rows.Next() {
		var tenant, id string
		var seq uint64
		var raw []byte
		if e = rows.Scan(&tenant, &id, &seq, &raw); e != nil {
			return nil, 0, e
		}
		var event Event
		if json.Unmarshal(raw, &event) != nil || event.TenantID != tenant || event.ID != id || seq != uint64(len(audit)+1) {
			return nil, 0, errors.New("invalid audit snapshot record")
		}
		if _, e = auditProjection(event); e != nil {
			return nil, 0, e
		}
		size += len(raw)
		if size > 64<<20 {
			return nil, 0, errors.New("snapshot too large")
		}
		audit = append(audit, append(json.RawMessage(nil), raw...))
	}
	if e = rows.Err(); e != nil {
		return nil, 0, e
	}
	raw, e := json.Marshal(snapshotV2{snapshotV2Format, 2, state, audit})
	if len(raw) > 64<<20 {
		return nil, 0, errors.New("snapshot too large")
	}
	if e == nil {
		e = validateSnapshotJSON(raw)
	}
	return raw, len(audit), e
}
func parseMySQLV2(raw []byte) (json.RawMessage, []json.RawMessage, []Event, error) {
	var envelope snapshotV2
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != 4 || envelope.Format != snapshotV2Format || envelope.Version != 2 || envelope.State == nil || envelope.Audit == nil {
		return nil, nil, nil, errors.New("snapshot v2 envelope invalid")
	}
	var st State
	if json.Unmarshal(envelope.State, &st) != nil || len(st.Audit) != 0 {
		return nil, nil, nil, errors.New("snapshot audit appears in both stores")
	}
	events := []Event{}
	seen := map[string]bool{}
	for _, row := range envelope.Audit {
		var event Event
		if json.Unmarshal(row, &event) != nil {
			return nil, nil, nil, errors.New("snapshot audit invalid")
		}
		if _, e := auditProjection(event); e != nil {
			return nil, nil, nil, e
		}
		key := fmt.Sprintf("%d:%s%s", len(event.TenantID), event.TenantID, event.ID)
		if seen[key] {
			return nil, nil, nil, errors.New("duplicate snapshot audit identity")
		}
		seen[key] = true
		events = append(events, event)
	}
	return envelope.State, envelope.Audit, events, nil
}

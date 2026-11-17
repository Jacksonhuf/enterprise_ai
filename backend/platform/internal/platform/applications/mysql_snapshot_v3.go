package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type snapshotV3 struct {
	Format     string            `json:"format"`
	Version    int               `json:"schema_version"`
	StateRoot  json.RawMessage   `json:"state_root"`
	Relational applicationLayout `json:"relational"`
	Audit      []json.RawMessage `json:"audit"`
}

func exportMySQLV3(ctx context.Context, tx *sql.Tx) ([]byte, int, error) {
	var version int
	var stateRoot []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &stateRoot); err != nil || version != 3 {
		return nil, 0, errors.New("MySQL v3 snapshot requires schema 3")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(stateRoot, &root) != nil || root == nil {
		return nil, 0, errors.New("invalid v3 state root")
	}
	if _, ok := root["apps"]; ok {
		return nil, 0, errors.New("v3 state root retains application authority")
	}
	if _, ok := root["checks"]; ok {
		return nil, 0, errors.New("v3 state root retains check authority")
	}
	layout, err := scanMySQLApplicationLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	if err = verifyMySQLApplicationProjections(ctx, tx, layout); err != nil {
		return nil, 0, err
	}
	if _, _, err = readMySQLApplicationV3(ctx, tx, false); err != nil {
		return nil, 0, err
	}
	audit, count, err := exportMySQLV3Audit(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	raw, err := json.Marshal(snapshotV3{snapshotV2Format, 3, stateRoot, layout, audit})
	if len(raw) > 64<<20 {
		return nil, 0, errors.New("snapshot too large")
	}
	if err == nil {
		err = validateSnapshotJSON(raw)
	}
	return raw, count, err
}

func exportMySQLV3Audit(ctx context.Context, tx *sql.Tx) ([]json.RawMessage, int, error) {
	if err := validateMySQLAuditSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT tenant_id,event_id,sequence_no,payload FROM cognara_fg_audit ORDER BY sequence_no`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	audit := []json.RawMessage{}
	for rows.Next() {
		var tenant, id string
		var seq uint64
		var raw []byte
		if err = rows.Scan(&tenant, &id, &seq, &raw); err != nil {
			return nil, 0, err
		}
		var event Event
		if json.Unmarshal(raw, &event) != nil || event.TenantID != tenant || event.ID != id || seq != uint64(len(audit)+1) {
			return nil, 0, errors.New("invalid audit snapshot record")
		}
		if _, err = auditProjection(event); err != nil {
			return nil, 0, err
		}
		audit = append(audit, append(json.RawMessage(nil), raw...))
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return audit, len(audit), nil
}

func parseMySQLV3(raw []byte) (json.RawMessage, applicationLayout, []json.RawMessage, []Event, error) {
	var envelope snapshotV3
	var fields map[string]json.RawMessage
	var zero applicationLayout
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != 5 || envelope.Format != snapshotV2Format || envelope.Version != 3 || envelope.StateRoot == nil || envelope.Audit == nil {
		return nil, zero, nil, nil, errors.New("snapshot v3 envelope invalid")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(envelope.StateRoot, &root) != nil || root == nil {
		return nil, zero, nil, nil, errors.New("snapshot v3 state root invalid")
	}
	if _, ok := root["apps"]; ok {
		return nil, zero, nil, nil, errors.New("snapshot v3 state root retains aggregate authority")
	}
	if _, ok := root["checks"]; ok {
		return nil, zero, nil, nil, errors.New("snapshot v3 state root retains aggregate authority")
	}
	events := []Event{}
	seen := map[string]bool{}
	for _, row := range envelope.Audit {
		var event Event
		if json.Unmarshal(row, &event) != nil {
			return nil, zero, nil, nil, errors.New("snapshot audit invalid")
		}
		if _, err := auditProjection(event); err != nil {
			return nil, zero, nil, nil, err
		}
		key := fmt.Sprintf("%d:%s%s", len(event.TenantID), event.TenantID, event.ID)
		if seen[key] {
			return nil, zero, nil, nil, errors.New("duplicate snapshot audit identity")
		}
		seen[key] = true
		events = append(events, event)
	}
	return envelope.StateRoot, envelope.Relational, envelope.Audit, events, nil
}

func stateFromV3Snapshot(stateRoot []byte, layout applicationLayout) (State, error) {
	var zero State
	joined, err := joinApplicationLayout(layout, time.Now().UTC())
	if err != nil {
		return zero, err
	}
	var root, joinedFields map[string]json.RawMessage
	if json.Unmarshal(stateRoot, &root) != nil || root == nil || json.Unmarshal(joined, &joinedFields) != nil {
		return zero, errors.New("snapshot v3 merge invalid")
	}
	for _, key := range []string{"apps", "checks"} {
		if value, ok := joinedFields[key]; ok {
			root[key] = value
		}
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return zero, err
	}
	var st State
	if json.Unmarshal(raw, &st) != nil || st.Apps == nil || st.Sessions == nil {
		return zero, errors.New("snapshot v3 state invalid")
	}
	initializeState(&st)
	return st, nil
}

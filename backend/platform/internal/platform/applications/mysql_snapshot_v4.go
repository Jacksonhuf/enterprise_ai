package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type snapshotV4 struct {
	Format     string            `json:"format"`
	Version    int               `json:"schema_version"`
	StateRoot  json.RawMessage   `json:"state_root"`
	Relational applicationLayout `json:"relational"`
	Grants     []grantLayoutRow  `json:"grants"`
	Audit      []json.RawMessage `json:"audit"`
}

func exportMySQLV4(ctx context.Context, tx *sql.Tx) ([]byte, int, error) {
	var version int
	var stateRoot []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &stateRoot); err != nil || version != 4 {
		return nil, 0, errors.New("MySQL v4 snapshot requires schema 4")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(stateRoot, &root) != nil || root == nil {
		return nil, 0, errors.New("invalid v4 state root")
	}
	if _, ok := root["apps"]; ok {
		return nil, 0, errors.New("v4 state root retains application authority")
	}
	if _, ok := root["checks"]; ok {
		return nil, 0, errors.New("v4 state root retains check authority")
	}
	if _, ok := root["grants"]; ok {
		return nil, 0, errors.New("v4 state root retains grant authority")
	}
	if err := validateMySQLApplicationSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	if err := validateMySQLGrantSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	layout, err := scanMySQLApplicationLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	if err = verifyMySQLApplicationProjections(ctx, tx, layout); err != nil {
		return nil, 0, err
	}
	grants, err := scanGrantLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	if _, _, err = readMySQLApplicationV4(ctx, tx, false); err != nil {
		return nil, 0, err
	}
	audit, count, err := exportMySQLV3Audit(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	raw, err := json.Marshal(snapshotV4{snapshotV2Format, 4, stateRoot, layout, grants, audit})
	if len(raw) > 64<<20 {
		return nil, 0, errors.New("snapshot too large")
	}
	if err == nil {
		err = validateSnapshotJSON(raw)
	}
	return raw, count, err
}

func parseMySQLV4(raw []byte) (json.RawMessage, applicationLayout, []grantLayoutRow, []json.RawMessage, []Event, error) {
	var envelope snapshotV4
	var fields map[string]json.RawMessage
	var zero applicationLayout
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != 6 || envelope.Format != snapshotV2Format || envelope.Version != 4 || envelope.StateRoot == nil || envelope.Audit == nil {
		return nil, zero, nil, nil, nil, errors.New("snapshot v4 envelope invalid")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(envelope.StateRoot, &root) != nil || root == nil {
		return nil, zero, nil, nil, nil, errors.New("snapshot v4 state root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants"} {
		if _, ok := root[key]; ok {
			return nil, zero, nil, nil, nil, errors.New("snapshot v4 state root retains aggregate authority")
		}
	}
	events := []Event{}
	seen := map[string]bool{}
	for _, row := range envelope.Audit {
		var event Event
		if json.Unmarshal(row, &event) != nil {
			return nil, zero, nil, nil, nil, errors.New("snapshot audit invalid")
		}
		if _, err := auditProjection(event); err != nil {
			return nil, zero, nil, nil, nil, err
		}
		key := fmt.Sprintf("%d:%s%s", len(event.TenantID), event.TenantID, event.ID)
		if seen[key] {
			return nil, zero, nil, nil, nil, errors.New("duplicate snapshot audit identity")
		}
		seen[key] = true
		events = append(events, event)
	}
	for _, row := range envelope.Grants {
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.ID != row.GrantID || g.TenantID != row.TenantID {
			return nil, zero, nil, nil, nil, errors.New("snapshot grant row invalid")
		}
	}
	return envelope.StateRoot, envelope.Relational, envelope.Grants, envelope.Audit, events, nil
}

func stateFromV4Snapshot(stateRoot []byte, layout applicationLayout, grantRows []grantLayoutRow) (State, error) {
	st, err := stateFromV3Snapshot(stateRoot, layout)
	if err != nil {
		return st, err
	}
	grants := make([]Grant, 0, len(grantRows))
	for _, row := range grantRows {
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.ID != row.GrantID || g.TenantID != row.TenantID {
			return State{}, errors.New("snapshot grant invalid")
		}
		grants = append(grants, g)
	}
	st.Grants = grants
	joined, err := joinGrantLayout(grantLayout{Root: stateRoot, Grants: grantRows, GrantsKind: "array"})
	if err != nil {
		return State{}, err
	}
	if _, err = planGrantSource(joined); err != nil {
		return State{}, err
	}
	return st, nil
}
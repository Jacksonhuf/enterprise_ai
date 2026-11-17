package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type snapshotV5 struct {
	Format     string             `json:"format"`
	Version    int                `json:"schema_version"`
	StateRoot  json.RawMessage    `json:"state_root"`
	Relational applicationLayout  `json:"relational"`
	Grants     []grantLayoutRow   `json:"grants"`
	Sessions   []sessionLayoutRow `json:"sessions"`
	Audit      []json.RawMessage  `json:"audit"`
}

func exportMySQLV5(ctx context.Context, tx *sql.Tx) ([]byte, int, error) {
	var version int
	var stateRoot []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &stateRoot); err != nil || version != 5 {
		return nil, 0, errors.New("MySQL v5 snapshot requires schema 5")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(stateRoot, &root) != nil || root == nil {
		return nil, 0, errors.New("invalid v5 state root")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions"} {
		if _, ok := root[key]; ok {
			return nil, 0, errors.New("v5 state root retains aggregate authority")
		}
	}
	if err := validateMySQLApplicationSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	if err := validateMySQLGrantSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	if err := validateMySQLSessionSchema(ctx, tx); err != nil {
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
	sessions, err := scanSessionLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	if _, _, err = readMySQLApplicationV5(ctx, tx, false); err != nil {
		return nil, 0, err
	}
	audit, count, err := exportMySQLV3Audit(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	raw, err := json.Marshal(snapshotV5{snapshotV2Format, 5, stateRoot, layout, grants, sessions, audit})
	if len(raw) > 64<<20 {
		return nil, 0, errors.New("snapshot too large")
	}
	if err == nil {
		err = validateSnapshotJSON(raw)
	}
	return raw, count, err
}

func parseMySQLV5(raw []byte) (json.RawMessage, applicationLayout, []grantLayoutRow, []sessionLayoutRow, []json.RawMessage, []Event, error) {
	var envelope snapshotV5
	var fields map[string]json.RawMessage
	var zero applicationLayout
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != 7 || envelope.Format != snapshotV2Format || envelope.Version != 5 || envelope.StateRoot == nil || envelope.Audit == nil {
		return nil, zero, nil, nil, nil, nil, errors.New("snapshot v5 envelope invalid")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(envelope.StateRoot, &root) != nil || root == nil {
		return nil, zero, nil, nil, nil, nil, errors.New("snapshot v5 state root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions"} {
		if _, ok := root[key]; ok {
			return nil, zero, nil, nil, nil, nil, errors.New("snapshot v5 state root retains aggregate authority")
		}
	}
	events := []Event{}
	seen := map[string]bool{}
	for _, row := range envelope.Audit {
		var event Event
		if json.Unmarshal(row, &event) != nil {
			return nil, zero, nil, nil, nil, nil, errors.New("snapshot audit invalid")
		}
		if _, err := auditProjection(event); err != nil {
			return nil, zero, nil, nil, nil, nil, err
		}
		key := fmt.Sprintf("%d:%s%s", len(event.TenantID), event.TenantID, event.ID)
		if seen[key] {
			return nil, zero, nil, nil, nil, nil, errors.New("duplicate snapshot audit identity")
		}
		seen[key] = true
		events = append(events, event)
	}
	for _, row := range envelope.Grants {
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.ID != row.GrantID || g.TenantID != row.TenantID {
			return nil, zero, nil, nil, nil, nil, errors.New("snapshot grant row invalid")
		}
	}
	for _, row := range envelope.Sessions {
		var s Session
		if json.Unmarshal(row.Payload, &s) != nil || s.ID != row.SessionID || s.TenantID != row.TenantID {
			return nil, zero, nil, nil, nil, nil, errors.New("snapshot session row invalid")
		}
		for _, msg := range row.Messages {
			var m Message
			if json.Unmarshal(msg.Payload, &m) != nil || m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return nil, zero, nil, nil, nil, nil, errors.New("snapshot session message row invalid")
			}
		}
	}
	return envelope.StateRoot, envelope.Relational, envelope.Grants, envelope.Sessions, envelope.Audit, events, nil
}

func stateFromV5Snapshot(stateRoot []byte, layout applicationLayout, grantRows []grantLayoutRow, sessionRows []sessionLayoutRow) (State, error) {
	st, err := stateFromV4Snapshot(stateRoot, layout, grantRows)
	if err != nil {
		return st, err
	}
	sessions := map[string]Session{}
	for _, row := range sessionRows {
		var header Session
		if json.Unmarshal(row.Payload, &header) != nil || header.ID != row.SessionID || header.TenantID != row.TenantID {
			return State{}, errors.New("snapshot session invalid")
		}
		messages := make([]Message, 0, len(row.Messages))
		for _, msg := range row.Messages {
			var m Message
			if json.Unmarshal(msg.Payload, &m) != nil || m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return State{}, errors.New("snapshot session message invalid")
			}
			messages = append(messages, m)
		}
		header.Messages = messages
		sessions[header.ID] = header
	}
	st.Sessions = sessions
	kind := "absent"
	if len(sessionRows) > 0 {
		kind = "object"
	}
	joined, err := joinSessionLayout(sessionLayout{Root: stateRoot, Sessions: sessionRows, SessionsKind: kind})
	if err != nil {
		return State{}, err
	}
	if _, err = planSessionSource(joined); err != nil {
		return State{}, err
	}
	return st, nil
}
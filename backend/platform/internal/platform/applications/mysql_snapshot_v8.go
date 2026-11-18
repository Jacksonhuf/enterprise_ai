package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type snapshotV8 struct {
	Format     string               `json:"format"`
	Version    int                  `json:"schema_version"`
	StateRoot  json.RawMessage      `json:"state_root"`
	Relational applicationLayout    `json:"relational"`
	Grants     []grantLayoutRow     `json:"grants"`
	Sessions   []sessionLayoutRow   `json:"sessions"`
	Settings   []settingsLayoutRow  `json:"settings"`
	Exceptions []exceptionLayoutRow `json:"exceptions"`
	Usage      []usageLayoutRow     `json:"usage"`
	Audit      []json.RawMessage    `json:"audit"`
}

func exportMySQLV8(ctx context.Context, tx *sql.Tx) ([]byte, int, error) {
	var version int
	var stateRoot []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &stateRoot); err != nil || version != 8 {
		return nil, 0, errors.New("MySQL v8 snapshot requires schema 8")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(stateRoot, &root) != nil || root == nil {
		return nil, 0, errors.New("invalid v8 state root")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions", "settings", "setting_revisions", "exceptions", "usage"} {
		if _, ok := root[key]; ok {
			return nil, 0, errors.New("v8 state root retains aggregate authority")
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
	if err := validateMySQLSettingsSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	if err := validateMySQLExceptionSchema(ctx, tx); err != nil {
		return nil, 0, err
	}
	if err := validateMySQLUsageSchema(ctx, tx); err != nil {
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
	settings, err := scanSettingsLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	exceptions, err := scanExceptionLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	usage, err := scanUsageLayout(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	if _, _, err = readMySQLApplicationV8(ctx, tx, false); err != nil {
		return nil, 0, err
	}
	audit, count, err := exportMySQLV3Audit(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	raw, err := json.Marshal(snapshotV8{snapshotV2Format, 8, stateRoot, layout, grants, sessions, settings, exceptions, usage, audit})
	if len(raw) > 64<<20 {
		return nil, 0, errors.New("snapshot too large")
	}
	if err == nil {
		err = validateSnapshotJSON(raw)
	}
	return raw, count, err
}

func parseMySQLV8(raw []byte) (json.RawMessage, applicationLayout, []grantLayoutRow, []sessionLayoutRow, []settingsLayoutRow, []exceptionLayoutRow, []usageLayoutRow, []json.RawMessage, []Event, error) {
	var envelope snapshotV8
	var fields map[string]json.RawMessage
	var zero applicationLayout
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != 10 || envelope.Format != snapshotV2Format || envelope.Version != 8 || envelope.StateRoot == nil || envelope.Audit == nil {
		return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot v8 envelope invalid")
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(envelope.StateRoot, &root) != nil || root == nil {
		return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot v8 state root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions", "settings", "setting_revisions", "exceptions", "usage"} {
		if _, ok := root[key]; ok {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot v8 state root retains aggregate authority")
		}
	}
	events := []Event{}
	seen := map[string]bool{}
	for _, row := range envelope.Audit {
		var event Event
		if json.Unmarshal(row, &event) != nil {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot audit invalid")
		}
		if _, err := auditProjection(event); err != nil {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, err
		}
		key := fmt.Sprintf("%d:%s%s", len(event.TenantID), event.TenantID, event.ID)
		if seen[key] {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("duplicate snapshot audit identity")
		}
		seen[key] = true
		events = append(events, event)
	}
	for _, row := range envelope.Grants {
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.ID != row.GrantID || g.TenantID != row.TenantID {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot grant row invalid")
		}
	}
	for _, row := range envelope.Sessions {
		var s Session
		if json.Unmarshal(row.Payload, &s) != nil || s.ID != row.SessionID || s.TenantID != row.TenantID {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot session row invalid")
		}
		for _, msg := range row.Messages {
			var m Message
			if json.Unmarshal(msg.Payload, &m) != nil || m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot session message row invalid")
			}
		}
	}
	for _, row := range envelope.Settings {
		owner, section, err := parseSettingKey(row.SettingKey)
		if err != nil || owner != row.OwnerID || section != row.Section {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot settings row invalid")
		}
		if !applicationPlanID(row.TenantID, 64) || row.Revision == "" {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot settings identity invalid")
		}
	}
	for _, row := range envelope.Exceptions {
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil || item["id"] != row.ExceptionID || item["tenant_id"] != row.TenantID {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot exception row invalid")
		}
		if row.Ordinal == 0 {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot exception ordinal invalid")
		}
	}
	for _, row := range envelope.Usage {
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil || item["event_id"] != row.EventID || item["tenant_id"] != row.TenantID {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot usage row invalid")
		}
		if row.Ordinal == 0 {
			return nil, zero, nil, nil, nil, nil, nil, nil, nil, errors.New("snapshot usage ordinal invalid")
		}
	}
	return envelope.StateRoot, envelope.Relational, envelope.Grants, envelope.Sessions, envelope.Settings, envelope.Exceptions, envelope.Usage, envelope.Audit, events, nil
}

func stateFromV8Snapshot(stateRoot []byte, layout applicationLayout, grantRows []grantLayoutRow, sessionRows []sessionLayoutRow, settingsRows []settingsLayoutRow, exceptionRows []exceptionLayoutRow, usageRows []usageLayoutRow) (State, error) {
	st, err := stateFromV7Snapshot(stateRoot, layout, grantRows, sessionRows, settingsRows, exceptionRows)
	if err != nil {
		return st, err
	}
	st.Usage = make([]map[string]any, 0, len(usageRows))
	for _, row := range usageRows {
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil {
			return State{}, errors.New("snapshot usage invalid")
		}
		st.Usage = append(st.Usage, item)
	}
	kind := "absent"
	if len(usageRows) > 0 {
		kind = "array"
	}
	joined, err := joinUsageLayout(usageLayout{Root: stateRoot, Usage: usageRows, UsageKind: kind})
	if err != nil {
		return State{}, err
	}
	if _, err = planUsageSource(joined); err != nil {
		return State{}, err
	}
	return st, nil
}

func applyMySQLUsageSnapshot(ctx context.Context, tx *sql.Tx, rows []usageLayoutRow) error {
	if err := validateMySQLUsageSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_usage`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("usage import target must be empty")
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range rows {
		meta, err := json.Marshal(map[string]any{"unit": row.Unit, "capability": row.Capability, "occurred_at": row.OccurredAt, "ordinal": row.Ordinal})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_usage(tenant_id,event_id,ordinal,unit,capability,occurred_at,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?)`,
			row.TenantID, row.EventID, row.Ordinal, row.Unit, row.Capability, row.OccurredAt, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}
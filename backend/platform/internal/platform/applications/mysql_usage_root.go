package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalUsageDomain = errors.New("usage changes require relational writers")

func stageMySQLUsageRoot(ctx context.Context, tx *sql.Tx) error {
	var version int
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &original); err != nil {
		return err
	}
	if version != 7 {
		return errors.New("usage root staging requires schema 7")
	}
	layout, err := splitUsageLayout(original)
	if err != nil {
		return err
	}
	if err = applyMySQLUsageLayout(ctx, tx, original); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	got, err := loadStagedMySQLUsageState(ctx, tx, layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("usage root handoff reconciliation failed")
	}
	return nil
}

func scanUsageLayout(ctx context.Context, q applicationSchemaQuery) ([]usageLayoutRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT tenant_id,event_id,ordinal,unit,capability,occurred_at,raw_payload FROM cognara_fg_usage ORDER BY ordinal`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layout []usageLayoutRow
	for rows.Next() {
		var tenant, id, unit, capability, occurred, raw []byte
		var ordinal uint64
		if err = rows.Scan(&tenant, &id, &ordinal, &unit, &capability, &occurred, &raw); err != nil {
			return nil, err
		}
		layout = append(layout, usageLayoutRow{
			TenantID: string(tenant), EventID: string(id), Ordinal: ordinal,
			Unit: string(unit), Capability: string(capability), OccurredAt: string(occurred), Payload: append(json.RawMessage(nil), raw...),
		})
	}
	return layout, rows.Err()
}

func loadStagedMySQLUsageState(ctx context.Context, tx *sql.Tx, layout usageLayout) ([]byte, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return nil, err
	}
	if version != 7 {
		return nil, errors.New("usage root staging requires schema 7")
	}
	rows, err := scanUsageLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(layout.Usage) {
		return nil, errors.New("usage staging row count mismatch")
	}
	byID := map[string]usageLayoutRow{}
	for _, row := range rows {
		byID[row.EventID] = row
	}
	for _, expected := range layout.Usage {
		row, ok := byID[expected.EventID]
		if !ok || row.TenantID != expected.TenantID || row.Ordinal != expected.Ordinal || !equalApplicationJSON(expected.Payload, row.Payload) {
			return nil, errors.New("staged usage row mismatch")
		}
	}
	root, err := currentUsageStagingRoot(ctx, tx)
	if err != nil {
		return nil, err
	}
	kind := layout.UsageKind
	if len(rows) > 0 {
		kind = "array"
	}
	return joinUsageLayout(usageLayout{Root: root, Usage: rows, UsageKind: kind})
}

func currentUsageStagingRoot(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return nil, errors.New("invalid usage staging root")
	}
	if _, ok := object["usage"]; ok {
		return nil, errors.New("usage staging root still contains usage")
	}
	return append([]byte(nil), root...), nil
}

func loadMySQLUsageFromTable(ctx context.Context, q applicationSchemaQuery) ([]map[string]any, error) {
	rows, err := scanUsageLayout(ctx, q)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil {
			return nil, errors.New("usage table payload invalid")
		}
		items = append(items, item)
	}
	return items, nil
}

func insertMySQLUsageRow(ctx context.Context, tx *sql.Tx, event map[string]any) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	id, _ := event["event_id"].(string)
	tenant, _ := event["tenant_id"].(string)
	if !applicationPlanID(id, 512) || !applicationPlanID(tenant, 64) {
		return errors.New("invalid usage identity")
	}
	unit, capability, occurred := usageProjectionDefaults(event)
	var ordinal uint64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal),0)+1 FROM cognara_fg_usage`).Scan(&ordinal); err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{"unit": unit, "capability": capability, "occurred_at": occurred, "ordinal": ordinal})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_usage(tenant_id,event_id,ordinal,unit,capability,occurred_at,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?)`,
		tenant, id, ordinal, unit, capability, occurred, string(meta), raw)
	return err
}

func applyMySQLUsageAppendDelta(ctx context.Context, tx *sql.Tx, before, after []map[string]any) error {
	beforeByID := map[string]json.RawMessage{}
	for _, item := range before {
		id, _ := item["event_id"].(string)
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		beforeByID[id] = raw
	}
	for _, item := range after {
		id, _ := item["event_id"].(string)
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if prev, ok := beforeByID[id]; ok {
			if !equalApplicationJSON(prev, raw) {
				return errRelationalUsageDomain
			}
			delete(beforeByID, id)
			continue
		}
		if err = insertMySQLUsageRow(ctx, tx, item); err != nil {
			return err
		}
	}
	if len(beforeByID) != 0 {
		return errRelationalUsageDomain
	}
	return nil
}
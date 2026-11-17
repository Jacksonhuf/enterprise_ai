package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalSettingsDomain = errors.New("settings changes require relational writers")

func stageMySQLSettingsRoot(ctx context.Context, tx *sql.Tx) error {
	var version int
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &original); err != nil {
		return err
	}
	if version != 5 {
		return errors.New("settings root staging requires schema 5")
	}
	st, _, err := readMySQLApplicationV5(ctx, tx, false)
	if err != nil {
		return err
	}
	tenant, err := dominantTenantID(st)
	if err != nil {
		return err
	}
	layout, err := splitSettingsLayout(original)
	if err != nil {
		return err
	}
	layout.InferredTenant = tenant
	for i := range layout.Settings {
		layout.Settings[i].TenantID = tenant
	}
	if err = applyMySQLSettingsLayout(ctx, tx, original, tenant); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	got, err := loadStagedMySQLSettingsState(ctx, tx, layout, tenant)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("settings root handoff reconciliation failed")
	}
	return nil
}

func scanSettingsLayout(ctx context.Context, q applicationSchemaQuery) ([]settingsLayoutRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT tenant_id,setting_key,owner_id,section,revision,raw_payload FROM cognara_fg_settings ORDER BY tenant_id,setting_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layout []settingsLayoutRow
	for rows.Next() {
		var tenant, key, owner, section, revision, raw []byte
		if err = rows.Scan(&tenant, &key, &owner, &section, &revision, &raw); err != nil {
			return nil, err
		}
		var envelope struct {
			Value json.RawMessage `json:"value"`
		}
		if json.Unmarshal(raw, &envelope) != nil {
			return nil, errors.New("settings table payload invalid")
		}
		layout = append(layout, settingsLayoutRow{
			TenantID: string(tenant), SettingKey: string(key), OwnerID: string(owner), Section: string(section),
			Revision: string(revision), Payload: append(json.RawMessage(nil), envelope.Value...),
		})
	}
	return layout, rows.Err()
}

func loadStagedMySQLSettingsState(ctx context.Context, tx *sql.Tx, layout settingsLayout, tenant string) ([]byte, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return nil, err
	}
	if version != 5 {
		return nil, errors.New("settings root staging requires schema 5")
	}
	rows, err := scanSettingsLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(layout.Settings) {
		return nil, errors.New("settings staging row count mismatch")
	}
	byKey := map[string]settingsLayoutRow{}
	for _, row := range rows {
		byKey[row.SettingKey] = row
	}
	for _, expected := range layout.Settings {
		row, ok := byKey[expected.SettingKey]
		if !ok || row.TenantID != expected.TenantID || !equalApplicationJSON(expected.Payload, row.Payload) || row.Revision != expected.Revision {
			return nil, errors.New("staged settings row mismatch")
		}
	}
	root, err := currentSettingsStagingRoot(ctx, tx)
	if err != nil {
		return nil, err
	}
	rebuilt := settingsLayout{
		Root: root, Settings: rows, SettingsKind: layout.SettingsKind, RevisionsKind: layout.RevisionsKind, InferredTenant: tenant,
	}
	return joinSettingsLayout(rebuilt)
}

func currentSettingsStagingRoot(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return nil, errors.New("invalid settings staging root")
	}
	if _, ok := object["settings"]; ok {
		return nil, errors.New("settings staging root still contains settings")
	}
	if _, ok := object["setting_revisions"]; ok {
		return nil, errors.New("settings staging root still contains setting_revisions")
	}
	return append([]byte(nil), root...), nil
}

func loadMySQLSettingsFromTable(ctx context.Context, q applicationSchemaQuery) (map[string]json.RawMessage, map[string]string, error) {
	rows, err := scanSettingsLayout(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	values := map[string]json.RawMessage{}
	revisions := map[string]string{}
	for _, row := range rows {
		values[row.SettingKey] = append(json.RawMessage(nil), row.Payload...)
		revisions[row.SettingKey] = row.Revision
	}
	return values, revisions, nil
}

func upsertMySQLSettingsRow(ctx context.Context, tx *sql.Tx, tenant, key string, value json.RawMessage, revision string) error {
	owner, section, err := parseSettingKey(key)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{"revision": revision, "owner_id": owner, "section": section})
	if err != nil {
		return err
	}
	raw, err := json.Marshal(map[string]any{"value": json.RawMessage(value), "revision": revision})
	if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_settings WHERE tenant_id=? AND setting_key=?`, tenant, key).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_settings(tenant_id,setting_key,owner_id,section,revision,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
			tenant, key, owner, section, revision, string(meta), raw)
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE cognara_fg_settings SET owner_id=?,section=?,revision=?,metadata=?,raw_payload=? WHERE tenant_id=? AND setting_key=?`,
		owner, section, revision, string(meta), raw, tenant, key)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("settings row unavailable")
	}
	return nil
}
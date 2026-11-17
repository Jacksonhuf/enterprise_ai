package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// SettingsRecord is the canonical key or its explicitly authorized legacy fallback.
type SettingsRecord struct {
	Value               json.RawMessage
	Revision            string
	Canonical           bool
	initializeValues    bool
	initializeRevisions bool
}
type SettingsPatch struct {
	Value    json.RawMessage
	Revision string
	Audit    []Event
}

func settingsJSONPath(section, key string) string {
	quoted, _ := json.Marshal(key)
	return "$." + section + "." + string(quoted)
}

// UpdateMySQLSettingsContext reads only the selected configuration, then updates
// two JSON paths and appends audit in one transaction. It does not take Store.mu.
// The existing state row still serializes database writes. Other backends/versions
// and noncanonical legacy envelopes fall back before invoking the callback.
func (s *Store) UpdateMySQLSettingsContext(parent context.Context, key, legacy string, mutate func(SettingsRecord) (*SettingsPatch, error)) (bool, error) {
	if s.dialect != "mysql" {
		return false, nil
	}
	if key == "" || mutate == nil {
		return true, errors.New("invalid settings selection")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return true, e
	}
	defer tx.Rollback()
	record, handled, e := readMySQLSettingsProjection(ctx, tx, key, legacy, true)
	if !handled || e != nil {
		return handled, e
	}
	patch, e := mutate(record)
	if e != nil {
		return true, e
	}
	if patch == nil {
		return true, tx.Commit()
	}
	var value map[string]string
	if json.Unmarshal(patch.Value, &value) != nil || value == nil || patch.Revision == "" {
		return true, errors.New("invalid settings patch")
	}
	if len(patch.Audit) > 0 {
		var last uint64
		if e = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); e != nil {
			return true, e
		}
		if e = insertMySQLAudit(ctx, tx, patch.Audit, last); e != nil {
			return true, e
		}
	}
	base := "body"
	if record.initializeValues {
		base = "JSON_SET(" + base + ",'$.settings',JSON_OBJECT())"
	}
	if record.initializeRevisions {
		base = "JSON_SET(" + base + ",'$.setting_revisions',JSON_OBJECT())"
	}
	var version int
	if e = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); e != nil {
		return true, e
	}
	if version == 6 {
		st, _, e := readMySQLApplicationV6(ctx, tx, false)
		if e != nil {
			return true, e
		}
		tenant, e := dominantTenantID(st)
		if e != nil {
			return true, e
		}
		if e = upsertMySQLSettingsRow(ctx, tx, tenant, key, patch.Value, patch.Revision); e != nil {
			return true, e
		}
		if _, _, e = readMySQLApplicationV6(ctx, tx, false); e != nil {
			return true, e
		}
		return true, tx.Commit()
	}
	if _, e = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=JSON_SET(`+base+`,?,CAST(? AS JSON),?,?) WHERE id=1`, settingsJSONPath("settings", key), string(patch.Value), settingsJSONPath("setting_revisions", key), patch.Revision); e != nil {
		return true, e
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return true, e
		}
	}
	return true, tx.Commit()
}

// One projection decoder keeps read/write legacy and version semantics identical.
func readMySQLSettingsProjection(ctx context.Context, tx *sql.Tx, key, legacy string, lock bool) (SettingsRecord, bool, error) {
	fallback := legacy
	if fallback == "" {
		fallback = key
	}
	var version int
	var keys, current, old, revision, audit, settings, revisions []byte
	query := `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,?),JSON_EXTRACT(body,?),JSON_EXTRACT(body,?),JSON_EXTRACT(body,'$.audit'),JSON_TYPE(JSON_EXTRACT(body,'$.settings')),JSON_TYPE(JSON_EXTRACT(body,'$.setting_revisions')) FROM cognara_fg_state WHERE id=1`
	if lock {
		query += " FOR UPDATE"
	}
	e := tx.QueryRowContext(ctx, query, settingsJSONPath("settings", key), settingsJSONPath("settings", fallback), settingsJSONPath("setting_revisions", key)).Scan(&version, &keys, &current, &old, &revision, &audit, &settings, &revisions)
	if e != nil {
		return SettingsRecord{}, true, e
	}
	if version == 1 {
		return SettingsRecord{}, false, nil
	}
	if version != 2 && !mySQLRelationalRuntimeVersion(version) {
		return SettingsRecord{}, true, errors.New("unsupported state schema")
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return SettingsRecord{}, true, e
		}
	}
	if version == 6 {
		st, _, e := readMySQLApplicationV6(ctx, tx, false)
		if e != nil {
			return SettingsRecord{}, true, e
		}
		record := SettingsRecord{Value: json.RawMessage(`{}`)}
		if value, ok := st.Settings[key]; ok {
			record.Canonical = true
			record.Value = append(json.RawMessage(nil), value...)
			record.Revision = st.SettingRevisions[key]
		} else if legacy != "" {
			if value, ok := st.Settings[legacy]; ok {
				record.Value = append(json.RawMessage(nil), value...)
				record.Revision = st.SettingRevisions[legacy]
			}
		}
		return record, true, nil
	}
	var names []string
	if json.Unmarshal(keys, &names) != nil {
		return SettingsRecord{}, true, errors.New("invalid state envelope")
	}
	for _, name := range names {
		for _, canonical := range []string{"settings", "setting_revisions", "audit"} {
			if name != canonical && strings.EqualFold(name, canonical) {
				if mySQLRelationalRuntimeVersion(version) {
					return SettingsRecord{}, true, errors.New("ambiguous schema3 settings aliases")
				}
				return SettingsRecord{}, false, nil
			}
		}
	}
	for _, kind := range [][]byte{settings, revisions} {
		if len(kind) > 0 && string(kind) != "OBJECT" && string(kind) != "NULL" {
			return SettingsRecord{}, true, errors.New("invalid settings envelope")
		}
	}
	var embedded []json.RawMessage
	if len(audit) > 0 && (json.Unmarshal(audit, &embedded) != nil || len(embedded) != 0) {
		return SettingsRecord{}, true, errors.New("ambiguous audit storage")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return SettingsRecord{}, true, e
	}
	record := SettingsRecord{Value: json.RawMessage(`{}`), Canonical: len(current) > 0, initializeValues: string(settings) != "OBJECT", initializeRevisions: string(revisions) != "OBJECT"}
	if record.Canonical {
		record.Value = current
	} else if legacy != "" && len(old) > 0 {
		record.Value = old
	}
	if len(revision) > 0 && string(revision) != "null" {
		if json.Unmarshal(revision, &record.Revision) != nil {
			return SettingsRecord{}, true, errors.New("invalid settings revision")
		}
	}
	return record, true, nil
}

// ReadMySQLSettingsContext uses a non-locking consistent snapshot and never Store.mu.
func (s *Store) ReadMySQLSettingsContext(parent context.Context, key, legacy string) (SettingsRecord, bool, error) {
	if s.dialect != "mysql" {
		return SettingsRecord{}, false, nil
	}
	if key == "" {
		return SettingsRecord{}, true, errors.New("invalid settings selection")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return SettingsRecord{}, true, e
	}
	defer tx.Rollback()
	record, handled, e := readMySQLSettingsProjection(ctx, tx, key, legacy, false)
	if !handled || e != nil {
		return SettingsRecord{}, handled, e
	}
	if e = tx.Commit(); e != nil {
		return SettingsRecord{}, true, e
	}
	return record, true, nil
}

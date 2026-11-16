package applications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// SettingsRecord is the canonical key or its explicitly authorized legacy fallback.
type SettingsRecord struct {
	Value     json.RawMessage
	Revision  string
	Canonical bool
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
	fallback := legacy
	if fallback == "" {
		fallback = key
	}
	var version int
	var keys, current, old, revision, audit, settings, revisions []byte
	e = tx.QueryRowContext(ctx, `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,?),JSON_EXTRACT(body,?),JSON_EXTRACT(body,?),JSON_EXTRACT(body,'$.audit'),JSON_TYPE(JSON_EXTRACT(body,'$.settings')),JSON_TYPE(JSON_EXTRACT(body,'$.setting_revisions')) FROM cognara_fg_state WHERE id=1 FOR UPDATE`, settingsJSONPath("settings", key), settingsJSONPath("settings", fallback), settingsJSONPath("setting_revisions", key)).Scan(&version, &keys, &current, &old, &revision, &audit, &settings, &revisions)
	if e != nil {
		return true, e
	}
	if version == 1 {
		return false, nil
	}
	if version != 2 {
		return true, errors.New("unsupported state schema")
	}
	var names []string
	if json.Unmarshal(keys, &names) != nil {
		return true, errors.New("invalid state envelope")
	}
	for _, name := range names {
		for _, canonical := range []string{"settings", "setting_revisions", "audit"} {
			if name != canonical && strings.EqualFold(name, canonical) {
				return false, nil
			}
		}
	}
	for _, kind := range [][]byte{settings, revisions} {
		if len(kind) > 0 && string(kind) != "OBJECT" && string(kind) != "NULL" {
			return true, errors.New("invalid settings envelope")
		}
	}
	var embedded []json.RawMessage
	if len(audit) > 0 && (json.Unmarshal(audit, &embedded) != nil || len(embedded) != 0) {
		return true, errors.New("ambiguous audit storage")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return true, e
	}
	record := SettingsRecord{Value: json.RawMessage(`{}`), Canonical: len(current) > 0}
	if record.Canonical {
		record.Value = current
	} else if legacy != "" && len(old) > 0 {
		record.Value = old
	}
	if len(revision) > 0 && string(revision) != "null" {
		if json.Unmarshal(revision, &record.Revision) != nil {
			return true, errors.New("invalid settings revision")
		}
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
	if string(settings) != "OBJECT" {
		base = "JSON_SET(" + base + ",'$.settings',JSON_OBJECT())"
	}
	if string(revisions) != "OBJECT" {
		base = "JSON_SET(" + base + ",'$.setting_revisions',JSON_OBJECT())"
	}
	if _, e = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=JSON_SET(`+base+`,?,CAST(? AS JSON),?,?) WHERE id=1`, settingsJSONPath("settings", key), string(patch.Value), settingsJSONPath("setting_revisions", key), patch.Revision); e != nil {
		return true, e
	}
	return true, tx.Commit()
}

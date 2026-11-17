package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalDraftConflict = errors.New("relational draft revision conflict")
var errRelationalAppUnavailable = errors.New("relational application unavailable")

// Preserve opaque object extensions. Unchanged known fields retain their raw
// representation; a changed array is explicitly replaced as a whole.
func mergeRelationalDraft(raw []byte, next Config) ([]byte, error) {
	var previous Config
	if err := json.Unmarshal(raw, &previous); err != nil {
		return nil, err
	}
	oldKnown, err := json.Marshal(previous)
	if err != nil {
		return nil, err
	}
	newKnown, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	var merge func([]byte, []byte, []byte) ([]byte, error)
	merge = func(source, old, new []byte) ([]byte, error) {
		var src, oldObj, newObj map[string]json.RawMessage
		object := json.Unmarshal(new, &newObj) == nil && newObj != nil
		if object {
			if bytes.Equal(bytes.TrimSpace(source), []byte("null")) {
				if equalApplicationJSON(old, new) {
					return append([]byte(nil), source...), nil
				}
				return append([]byte(nil), new...), nil
			}
			if json.Unmarshal(source, &src) != nil || src == nil || json.Unmarshal(old, &oldObj) != nil {
				return nil, errors.New("invalid draft object")
			}
			keys := make([]string, 0, len(newObj))
			for key := range newObj {
				keys = append(keys, key)
			}
			if !canonicalFields(src, keys) {
				return nil, errors.New("ambiguous draft fields")
			}
			for key, value := range newObj {
				original, exists := src[key]
				if !exists {
					if equalApplicationJSON(oldObj[key], value) {
						continue
					}
					src[key] = value
					continue
				}
				merged, e := merge(original, oldObj[key], value)
				if e != nil {
					return nil, e
				}
				src[key] = merged
			}
			return json.Marshal(src)
		}
		if equalApplicationJSON(old, new) {
			return append([]byte(nil), source...), nil
		}
		return append([]byte(nil), new...), nil
	}
	return merge(raw, oldKnown, newKnown)
}

// Internal persistence primitive, not an HTTP authorization boundary. All callers
// must supply a trusted tenant/actor after permission checks. Startup stays gated.
// Retain the state-row barrier until remaining aggregate domains are migrated.
func saveMySQLV3Draft(ctx context.Context, db *sql.DB, tenant, id, actor, expected string, next Config) (string, error) {
	if tenant == "" || id == "" || actor == "" || expected == "" {
		return "", errors.New("invalid relational draft request")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
		return "", err
	}
	if version != 3 {
		return "", errors.New("relational draft writer requires schema 3")
	}
	st, _, err := readMySQLApplicationV3(ctx, tx, false)
	if err != nil {
		return "", err
	}
	a, ok := st.Apps[id]
	if !ok || a.TenantID != tenant {
		return "", errRelationalAppUnavailable
	}
	if a.Revision != expected {
		return "", errRelationalDraftConflict
	}
	var draft, header []byte
	if err = tx.QueryRowContext(ctx, `SELECT d.raw_payload,a.raw_payload FROM cognara_fg_application_drafts d JOIN cognara_fg_applications a ON a.tenant_id=d.tenant_id AND a.app_id=d.app_id WHERE d.tenant_id=? AND d.app_id=? FOR UPDATE`, tenant, id).Scan(&draft, &header); err != nil {
		return "", err
	}
	merged, err := mergeRelationalDraft(draft, next)
	if err != nil {
		return "", err
	}
	revision, updated := ID("draft"), Now()
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(header, &fields); err != nil {
		return "", err
	}
	fields["draft_revision"], _ = json.Marshal(revision)
	fields["updated_at"], _ = json.Marshal(updated)
	header, err = json.Marshal(fields)
	if err != nil {
		return "", err
	}
	result, err := tx.ExecContext(ctx, `UPDATE cognara_fg_application_drafts SET revision=?,config=?,raw_payload=? WHERE tenant_id=? AND app_id=? AND revision=?`, revision, string(merged), merged, tenant, id, expected)
	if err != nil {
		return "", err
	}
	if n, e := result.RowsAffected(); e != nil {
		return "", e
	} else if n != 1 {
		return "", errRelationalDraftConflict
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_applications SET updated_at=?,metadata=?,raw_payload=? WHERE tenant_id=? AND app_id=?`, updated, string(header), header, tenant, id); err != nil {
		return "", err
	}
	var last uint64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
		return "", err
	}
	if err = insertMySQLAudit(ctx, tx, []Event{{ID: ID("event"), TenantID: tenant, ActorID: actor, OccurredAt: updated, Action: "application.draft.save", ResourceID: id, Outcome: "success"}}, last); err != nil {
		return "", err
	}
	if _, _, err = readMySQLApplicationV3(ctx, tx, false); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return revision, nil
}

package applications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// AppendMySQLAuditContext records audit-only changes without rewriting body.
// The state lock preserves ordering with ordinary aggregate/settings mutations.
func (s *Store) AppendMySQLAuditContext(parent context.Context, events []Event) (bool, error) {
	return s.AppendMySQLObservabilityContext(parent, events, nil)
}

// AppendMySQLObservabilityContext atomically appends audit and an optional OPEN
// failure exception, without decoding or serializing unrelated business domains.
func (s *Store) AppendMySQLObservabilityContext(parent context.Context, events []Event, exception map[string]any) (bool, error) {
	if s.dialect != "mysql" {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return true, e
	}
	defer tx.Rollback()
	var version int
	var keys, audit, exceptionsKind []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,'$.audit'),JSON_TYPE(JSON_EXTRACT(body,'$.exceptions')) FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &keys, &audit, &exceptionsKind); e != nil {
		return true, e
	}
	if version == 1 {
		return false, nil
	}
	if version != 2 && !mySQLRelationalRuntimeVersion(version) {
		return true, errors.New("unsupported state schema")
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return true, e
		}
	}
	var names []string
	if json.Unmarshal(keys, &names) != nil {
		return true, errors.New("invalid state envelope")
	}
	for _, name := range names {
		if (name != "audit" && strings.EqualFold(name, "audit")) || (exception != nil && name != "exceptions" && strings.EqualFold(name, "exceptions")) {
			if mySQLRelationalRuntimeVersion(version) {
				return true, errors.New("ambiguous schema3 audit aliases")
			}
			return false, nil
		}
	}
	var embedded []json.RawMessage
	if len(audit) > 0 && (json.Unmarshal(audit, &embedded) != nil || len(embedded) > 0) {
		return true, errors.New("ambiguous audit storage")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return true, e
	}
	var exceptionRaw []byte
	if exception != nil {
		id, _ := exception["id"].(string)
		revision, _ := exception["revision"].(string)
		tenant, ok := exception["tenant_id"].(string)
		if len(events) == 0 || id == "" || revision == "" || !ok || tenant != events[0].TenantID || exception["status"] != "OPEN" {
			return true, errors.New("invalid failure exception")
		}
		if len(exceptionsKind) > 0 && string(exceptionsKind) != "ARRAY" && string(exceptionsKind) != "NULL" {
			return true, errors.New("invalid exception storage")
		}
		exceptionRaw, e = json.Marshal(exception)
		if e != nil {
			return true, errors.New("invalid failure exception JSON")
		}
	}
	if len(events) > 0 {
		var last uint64
		if e = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); e != nil {
			return true, e
		}
		if e = insertMySQLAudit(ctx, tx, events, last); e != nil {
			return true, e
		}
	}
	if exception != nil {
		if version == 7 {
			if e = validateMySQLExceptionSchema(ctx, tx); e != nil {
				return true, e
			}
			if e = insertMySQLExceptionRow(ctx, tx, exception); e != nil {
				return true, e
			}
		} else {
			query := `UPDATE cognara_fg_state SET body=JSON_ARRAY_APPEND(body,'$.exceptions',CAST(? AS JSON)) WHERE id=1`
			if string(exceptionsKind) != "ARRAY" {
				query = `UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',JSON_ARRAY(CAST(? AS JSON))) WHERE id=1`
			}
			if _, e = tx.ExecContext(ctx, query, string(exceptionRaw)); e != nil {
				return true, e
			}
		}
	}
	if mySQLRelationalRuntimeVersion(version) {
		if _, _, e = readMySQLApplicationRelational(ctx, tx, false, version); e != nil {
			return true, e
		}
	}
	return true, tx.Commit()
}

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
	var keys, audit []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,JSON_KEYS(body),JSON_EXTRACT(body,'$.audit') FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &keys, &audit); e != nil {
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
		if name != "audit" && strings.EqualFold(name, "audit") {
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
	if len(events) > 0 {
		var last uint64
		if e = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); e != nil {
			return true, e
		}
		if e = insertMySQLAudit(ctx, tx, events, last); e != nil {
			return true, e
		}
	}
	return true, tx.Commit()
}

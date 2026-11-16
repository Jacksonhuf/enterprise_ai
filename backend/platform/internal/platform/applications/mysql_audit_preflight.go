package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type AuditMigrationReport struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	AuditEvents         uint64 `json:"audit_events"`
	Status              string `json:"status"`
	ReadOnly            bool   `json:"read_only"`
}

// CheckMySQLAuditMigration never initializes, seeds, locks for update or writes.
// A successful check describes one snapshot, not permission to skip apply-time checks.
func CheckMySQLAuditMigration(parent context.Context, dsn string) (AuditMigrationReport, error) {
	var report AuditMigrationReport
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL migration configuration")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return report, errors.New("MySQL migration connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if e = validateMySQLStateSchema(ctx, db, cfg.DBName); e != nil {
		return report, errors.New("MySQL migration source incompatible")
	}
	tx, e := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return report, errors.New("MySQL migration check transaction unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); e != nil {
		return report, errors.New("MySQL migration state unavailable")
	}
	if version != 1 && version != 2 {
		return report, errors.New("MySQL migration version unsupported")
	}
	var count uint64
	if version == 1 {
		st, _, err := auditMigrationSource(raw)
		if err != nil {
			return report, err
		}
		count = uint64(len(st.Audit))
		var tables int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_audit'`).Scan(&tables); e != nil {
			return report, errors.New("MySQL audit target unavailable")
		}
		if tables != 0 {
			if e = validateMySQLAuditSchema(ctx, tx); e != nil {
				return report, errors.New("MySQL audit table incompatible")
			}
			var existing uint64
			if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_audit`).Scan(&existing); e != nil || existing != 0 {
				return report, errors.New("MySQL audit migration target must be empty")
			}
		}
	} else {
		var st State
		var object map[string]json.RawMessage
		if len(raw) > 64<<20 || json.Unmarshal(raw, &object) != nil || object == nil || json.Unmarshal(raw, &st) != nil || len(st.Audit) != 0 {
			return report, errors.New("ambiguous audit storage")
		}
		if e = validateMySQLAuditSchema(ctx, tx); e != nil {
			return report, errors.New("MySQL audit table incompatible")
		}
		count, e = verifyMySQLAuditStream(ctx, tx)
		if e != nil {
			return report, errors.New("MySQL audit records invalid or unavailable")
		}
	}
	if e = tx.Commit(); e != nil {
		return report, errors.New("MySQL migration check transaction failed")
	}
	status := "READY"
	if version == 2 {
		status = "ALREADY_APPLIED"
	}
	return AuditMigrationReport{version, 2, count, status, true}, nil
}

// Shared by preflight and apply so rejected data cannot pass the read-only plan.
func auditMigrationSource(raw []byte) (State, map[string]json.RawMessage, error) {
	var st State
	var object map[string]json.RawMessage
	if len(raw) > 64<<20 || json.Unmarshal(raw, &object) != nil || object == nil || json.Unmarshal(raw, &st) != nil {
		return st, nil, errors.New("MySQL migration state invalid or too large")
	}
	for key := range object {
		if strings.EqualFold(key, "audit") && key != "audit" {
			return st, nil, errors.New("ambiguous audit field; source retained")
		}
	}
	if audit, ok := object["audit"]; ok {
		decoder := json.NewDecoder(bytes.NewReader(audit))
		decoder.DisallowUnknownFields()
		var checked []Event
		if decoder.Decode(&checked) != nil {
			return st, nil, errors.New("audit contains unsupported fields; source retained")
		}
	}
	seen := map[[2]string]bool{}
	for _, event := range st.Audit {
		if _, e := auditProjection(event); e != nil {
			return st, nil, errors.New("audit event invalid; source retained")
		}
		key := [2]string{event.TenantID, event.ID}
		if seen[key] {
			return st, nil, errors.New("duplicate audit identity; source retained")
		}
		seen[key] = true
	}
	return st, object, nil
}

func verifyMySQLAuditStream(ctx context.Context, tx *sql.Tx) (uint64, error) {
	rows, e := tx.QueryContext(ctx, `SELECT tenant_id,event_id,sequence_no,payload FROM cognara_fg_audit ORDER BY sequence_no`)
	if e != nil {
		return 0, e
	}
	defer rows.Close()
	var count uint64
	for rows.Next() {
		var tenant, id string
		var seq uint64
		var raw []byte
		if e = rows.Scan(&tenant, &id, &seq, &raw); e != nil {
			return 0, e
		}
		var event Event
		if count == ^uint64(0) || json.Unmarshal(raw, &event) != nil || event.TenantID != tenant || event.ID != id || seq != count+1 {
			return 0, errors.New("invalid relational audit record")
		}
		if _, e = auditProjection(event); e != nil {
			return 0, e
		}
		count++
	}
	return count, rows.Err()
}

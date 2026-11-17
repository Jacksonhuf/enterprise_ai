package applications

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/go-sql-driver/mysql"
	"io"
	"os"
	"time"
)

// ImportMySQLSnapshot inserts a verified snapshot into an empty transitional state table.
// No existing record is ever updated. DDL may remain if a later transaction fails.
func ImportMySQLSnapshot(parent context.Context, dsn, path, expectedSHA string, sourceVersion int) (SnapshotReport, error) {
	var report SnapshotReport
	if sourceVersion != 1 && sourceVersion != 2 && sourceVersion != 3 {
		return report, errors.New("snapshot schema version unsupported")
	}
	expected, e := hex.DecodeString(expectedSHA)
	if e != nil || len(expected) != sha256.Size {
		return report, errors.New("expected SHA-256 is required")
	}
	file, e := os.Open(path)
	if e != nil {
		return report, errors.New("snapshot source unavailable")
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return report, errors.New("snapshot source must be a regular file")
	}
	raw, e := io.ReadAll(io.LimitReader(file, (64<<20)+1))
	if e != nil || len(raw) == 0 || len(raw) > 64<<20 {
		return report, errors.New("snapshot size unsupported")
	}
	sum := sha256.Sum256(raw)
	if !bytes.Equal(sum[:], expected) {
		return report, errors.New("snapshot checksum mismatch")
	}
	if e = validateSnapshotJSON(raw); e != nil {
		return report, errors.New("snapshot JSON invalid or ambiguous")
	}
	stateRaw := json.RawMessage(raw)
	var auditPayloads []json.RawMessage
	var auditEvents []Event
	var layout applicationLayout
	if sourceVersion == 2 {
		stateRaw, auditPayloads, auditEvents, e = parseMySQLV2(raw)
		if e != nil {
			return report, e
		}
	}
	if sourceVersion == 3 {
		stateRaw, layout, auditPayloads, auditEvents, e = parseMySQLV3(raw)
		if e != nil {
			return report, e
		}
	}
	var object map[string]json.RawMessage
	var state State
	if sourceVersion == 3 {
		state, e = stateFromV3Snapshot(stateRaw, layout)
		if e != nil {
			return report, errors.New("snapshot structure invalid")
		}
	} else if json.Unmarshal(stateRaw, &object) != nil || object == nil || json.Unmarshal(stateRaw, &state) != nil || state.Apps == nil || state.Sessions == nil {
		return report, errors.New("snapshot structure invalid")
	}
	for id, a := range state.Apps {
		if id == "" || a.ID != id || a.TenantID == "" || a.OwnerID == "" {
			return report, errors.New("snapshot application binding invalid")
		}
	}
	for id, s := range state.Sessions {
		a, ok := state.Apps[s.AppID]
		if id == "" || s.ID != id || s.OwnerID == "" || !ok || s.TenantID != a.TenantID {
			return report, errors.New("snapshot session binding invalid")
		}
		for _, m := range s.Messages {
			if m.SessionID != id || m.ID == "" {
				return report, errors.New("snapshot message binding invalid")
			}
		}
		report.Messages += len(s.Messages)
	}
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL import configuration")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return report, errors.New("MySQL import connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if _, e = db.ExecContext(ctx, mysqlStateDDL); e != nil {
		return report, errors.New("MySQL import schema initialization failed")
	}
	if e = validateMySQLStateSchema(ctx, db, cfg.DBName); e != nil {
		return report, errors.New("MySQL import schema incompatible")
	}
	if sourceVersion == 2 || sourceVersion == 3 {
		if _, e = db.ExecContext(ctx, mysqlAuditDDL); e != nil {
			return report, errors.New("MySQL audit import schema unavailable")
		}
	}
	if sourceVersion == 3 {
		if e = createMySQLApplicationSchema(ctx, db); e != nil {
			return report, errors.New("MySQL application import schema unavailable")
		}
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return report, errors.New("MySQL import transaction unavailable")
	}
	defer tx.Rollback()
	// The singleton primary key arbitrates concurrent importers and startup seeders.
	if _, e = tx.ExecContext(ctx, `INSERT INTO cognara_fg_state(id,schema_version,body) VALUES(1,?,?)`, sourceVersion, string(stateRaw)); e != nil {
		return report, errors.New("MySQL import refused; target must contain no state record")
	}
	if sourceVersion == 3 {
		if e = applyMySQLApplicationLayout(ctx, tx, layout); e != nil {
			return report, errors.New("MySQL v3 relational import rejected")
		}
	}
	if sourceVersion == 2 || sourceVersion == 3 {
		if e = validateMySQLAuditSchema(ctx, tx); e != nil {
			return report, errors.New("MySQL audit import schema incompatible")
		}
		var count int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_audit`).Scan(&count); e != nil || count != 0 {
			return report, errors.New("MySQL audit import target must be empty")
		}
		if e = insertMySQLAuditPayloads(ctx, tx, auditEvents, 0, auditPayloads); e != nil {
			return report, errors.New("MySQL audit import rejected")
		}
	}
	if sourceVersion == 3 {
		if _, _, e = readMySQLApplicationV3(ctx, tx, true); e != nil {
			return report, errors.New("MySQL v3 import reconciliation failed")
		}
	}
	if e = tx.Commit(); e != nil {
		return report, errors.New("MySQL import commit not confirmed; inspect target before retry")
	}
	report.SourceSchemaVersion = sourceVersion
	report.AuditEvents = len(state.Audit)
	if sourceVersion == 2 || sourceVersion == 3 {
		report.AuditEvents = len(auditEvents)
	}
	report.SHA256 = hex.EncodeToString(sum[:])
	report.Bytes = len(raw)
	report.Applications = len(state.Apps)
	report.Sessions = len(state.Sessions)
	return report, nil
}

// Reject duplicate object keys at every level, excessive nesting and trailing values.
func validateSnapshotJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 256 {
			return errors.New("too deep")
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return e
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return errors.New("duplicate key")
				}
				seen[key] = true
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
		default:
			return errors.New("unexpected delimiter")
		}
		_, e = d.Token()
		return e
	}
	if e := walk(0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

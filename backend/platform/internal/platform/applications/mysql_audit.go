package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed migrations/001_audit.sql
var mysqlAuditDDL string

// MigrateMySQLAudit explicitly moves v1 audit records into a relational table.
// The DDL may survive a failed migration. State and audit rows commit together.
func MigrateMySQLAudit(parent context.Context, dsn string) error {
	cfg, e := mysql.ParseDSN(dsn)
	if e != nil || cfg.DBName == "" || cfg.MultiStatements {
		return errors.New("invalid MySQL migration configuration")
	}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		return errors.New("MySQL migration connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if e = validateMySQLStateSchema(ctx, db, cfg.DBName); e != nil {
		return errors.New("MySQL migration source incompatible")
	}
	if _, e = db.ExecContext(ctx, mysqlAuditDDL); e != nil {
		return errors.New("MySQL audit schema initialization failed")
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return errors.New("MySQL migration transaction unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if e = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &raw); e != nil {
		return errors.New("MySQL migration state unavailable")
	}
	if version != 1 && version != 2 {
		return errors.New("MySQL migration version unsupported")
	}
	if e = validateMySQLAuditSchema(ctx, tx); e != nil {
		return errors.New("MySQL audit table incompatible")
	}
	if version == 2 {
		_, _, e = readStateRow(ctx, tx, false, true, true)
		if e != nil {
			return errors.New("MySQL audit migration verification failed")
		}
		return nil
	}
	// Refuse to merge with unrelated existing rows.
	var count int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_audit`).Scan(&count); e != nil || count != 0 {
		return errors.New("MySQL audit migration target must be empty")
	}
	st, object, e := auditMigrationSource(raw)
	if e != nil {
		return e
	}
	if e = appendMySQLAudit(ctx, tx, nil, st.Audit); e != nil {
		return errors.New("MySQL audit migration rejected; source retained")
	}
	// Preserve unknown aggregate fields during the one-time migration.
	delete(object, "audit")
	next, e := json.Marshal(object)
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET schema_version=2,body=? WHERE id=1`, string(next)); e != nil {
		return errors.New("MySQL migration version update failed")
	}
	if e = tx.Commit(); e != nil {
		return errors.New("MySQL migration commit not confirmed; inspect state before retry")
	}
	return nil
}
func loadMySQLAudit(ctx context.Context, tx *sql.Tx) ([]Event, error) {
	rows, e := tx.QueryContext(ctx, `SELECT tenant_id,event_id,sequence_no,payload FROM cognara_fg_audit ORDER BY sequence_no`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var tenant, id string
		var seq uint64
		var raw []byte
		if e = rows.Scan(&tenant, &id, &seq, &raw); e != nil {
			return nil, e
		}
		var event Event
		if json.Unmarshal(raw, &event) != nil || event.TenantID != tenant || event.ID != id || seq != uint64(len(events)+1) {
			return nil, errors.New("invalid relational audit record")
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
func appendMySQLAudit(ctx context.Context, tx *sql.Tx, previous, next []Event) error {
	if len(next) < len(previous) {
		return errors.New("audit records are append-only")
	}
	for i, event := range previous {
		if !reflect.DeepEqual(event, next[i]) {
			return errors.New("audit records are immutable")
		}
	}
	return insertMySQLAudit(ctx, tx, next[len(previous):], uint64(len(previous)))
}
func insertMySQLAudit(ctx context.Context, tx *sql.Tx, events []Event, last uint64) error {
	return insertMySQLAuditPayloads(ctx, tx, events, last, nil)
}
func insertMySQLAuditPayloads(ctx context.Context, tx *sql.Tx, events []Event, last uint64, payloads []json.RawMessage) error {
	if payloads != nil && len(payloads) != len(events) {
		return errors.New("invalid audit payload count")
	}
	for i, event := range events {
		if uint64(i)+1 > ^uint64(0)-last {
			return errors.New("audit sequence exhausted")
		}
		occurredColumn, e := auditProjection(event)
		if e != nil {
			return e
		}

		raw, e := json.Marshal(event)
		if e != nil {
			return e
		}
		if payloads != nil {
			raw = payloads[i]
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO cognara_fg_audit(tenant_id,event_id,sequence_no,occurred_at,actor_id,action,resource_id,payload) VALUES(?,?,?,?,?,?,?,?)`, event.TenantID, event.ID, last+uint64(i)+1, occurredColumn, event.ActorID, event.Action, event.ResourceID, string(raw)); e != nil {
			return e
		}
	}
	return nil
}

func validateMySQLAuditSchema(ctx context.Context, tx *sql.Tx) error {
	var engine string
	if e := tx.QueryRowContext(ctx, `SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_audit'`).Scan(&engine); e != nil {
		return e
	}
	if engine != "InnoDB" {
		return errors.New("audit engine incompatible")
	}
	expected := map[string]string{"tenant_id": "varbinary(64)", "event_id": "varbinary(512)", "sequence_no": "bigint unsigned", "occurred_at": "datetime(6)", "actor_id": "varchar(255)", "action": "varchar(255)", "resource_id": "varchar(255)", "payload": "json"}
	rows, e := tx.QueryContext(ctx, `SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_audit'`)
	if e != nil {
		return e
	}
	count := 0
	for rows.Next() {
		var name, kind, nullable, extra string
		if e = rows.Scan(&name, &kind, &nullable, &extra); e != nil {
			rows.Close()
			return e
		}
		wantNullable := "NO"
		if name == "occurred_at" {
			wantNullable = "YES"
		}
		if expected[name] != kind || nullable != wantNullable || extra != "" {
			rows.Close()
			return errors.New("audit column incompatible")
		}
		count++
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if count != len(expected) {
		return errors.New("audit columns incomplete")
	}
	indexes := map[string]string{"PRIMARY:1": "tenant_id:0", "PRIMARY:2": "event_id:0", "audit_sequence:1": "sequence_no:0", "audit_tenant_sequence:1": "tenant_id:1", "audit_tenant_sequence:2": "sequence_no:1"}
	rows, e = tx.QueryContext(ctx, `SELECT INDEX_NAME,SEQ_IN_INDEX,COLUMN_NAME,NON_UNIQUE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_audit'`)
	if e != nil {
		return e
	}
	count = 0
	for rows.Next() {
		var name, column string
		var position, nonunique int
		if e = rows.Scan(&name, &position, &column, &nonunique); e != nil {
			rows.Close()
			return e
		}
		key := fmt.Sprintf("%s:%d", name, position)
		if indexes[key] != fmt.Sprintf("%s:%d", column, nonunique) {
			rows.Close()
			return errors.New("audit index incompatible")
		}
		count++
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if count != len(indexes) {
		return errors.New("audit indexes incomplete")
	}
	return nil
}

func auditProjection(event Event) (any, error) {
	if event.ID == "" || len(event.TenantID) > 64 || len(event.ID) > 512 || utf8.RuneCountInString(event.ActorID) > 255 || utf8.RuneCountInString(event.Action) > 255 || utf8.RuneCountInString(event.ResourceID) > 255 {
		return nil, errors.New("invalid audit identity")
	}
	occurred, e := time.Parse(time.RFC3339Nano, strings.ToUpper(event.OccurredAt))
	if e != nil || occurred.Year() < 1 {
		return nil, errors.New("audit timestamp invalid")
	}
	var occurredColumn any
	utc := occurred.UTC()
	// The existing contract permits years outside MySQL DATETIME's range.
	// Preserve the exact source timestamp in payload; leave this projection NULL.
	if utc.Year() >= 1000 && utc.Year() <= 9999 {
		occurredColumn = utc.Format("2006-01-02 15:04:05.000000")
	}
	return occurredColumn, nil
}

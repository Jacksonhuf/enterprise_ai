package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// SessionMigrationPlan is a read-only snapshot report for schema4 session split.
type SessionMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Sessions            int    `json:"sessions"`
	Messages            int    `json:"messages"`
	Tenants             int    `json:"tenants"`
	Applications        int    `json:"applications"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLSessions requires schema4 with application and grant relational tables present.
func PlanMySQLSessions(parent context.Context, dsn string) (SessionMigrationPlan, error) {
	var report SessionMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL session plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL session plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL session plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL session plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL session plan source unavailable")
	}
	if version != 4 {
		return report, errors.New("MySQL session plan requires schema 4; complete grant migration first")
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL session plan application schema incompatible")
	}
	if err = validateMySQLGrantSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL session plan grant schema incompatible")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL session plan audit schema incompatible")
	}
	report, err = planSessionSource(raw)
	if err != nil {
		return SessionMigrationPlan{}, err
	}
	if err = verifySessionLayout(raw); err != nil {
		return SessionMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('cognara_fg_sessions','cognara_fg_session_messages')`).Scan(&targets); err != nil || targets != 0 {
		return SessionMigrationPlan{}, errors.New("MySQL session plan reserved target present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return SessionMigrationPlan{}, errors.New("MySQL session plan snapshot not confirmed")
	}
	return report, nil
}

func planSessionSource(root []byte) (SessionMigrationPlan, error) {
	var result SessionMigrationPlan
	if _, err := splitSessionLayout(root); err != nil {
		return result, err
	}
	var object map[string]json.RawMessage
	var state State
	if len(root) > 64<<20 || json.Unmarshal(root, &object) != nil || object == nil || json.Unmarshal(root, &state) != nil {
		return result, errors.New("MySQL session plan source invalid or too large")
	}
	if !canonicalFields(object, []string{"sessions", "settings", "usage", "receipts", "exceptions", "audit"}) {
		return result, errors.New("MySQL session plan ambiguous source fields")
	}
	value, ok := object["sessions"]
	if !ok {
		result.SourceSchemaVersion, result.TargetSchemaVersion = 4, 5
		result.Status, result.ReadOnly = "PLAN_VALIDATED", true
		result.ApplySupported = true
		sum := sha256.Sum256(root)
		result.SourceSHA256 = hex.EncodeToString(sum[:])
		result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
		return result, nil
	}
	if bytesTrim(value) == "null" {
		return result, errors.New("MySQL session plan null session container unsupported")
	}
	var items map[string]json.RawMessage
	if json.Unmarshal(value, &items) != nil || items == nil {
		return result, errors.New("MySQL session plan sessions must be an object")
	}
	tenants := map[string]bool{}
	apps := map[string]bool{}
	seenSessions := map[string]bool{}
	seenMessages := map[string]bool{}
	for key, raw := range items {
		var row map[string]json.RawMessage
		var s Session
		if json.Unmarshal(raw, &row) != nil || row == nil || json.Unmarshal(raw, &s) != nil {
			return result, errors.New("MySQL session plan session invalid")
		}
		if !canonicalFields(row, []string{"session_id", "app_id", "tenant_id", "owner_id", "title", "created_at", "updated_at", "revision", "title_customized", "messages"}) ||
			key != s.ID || !applicationPlanID(s.ID, 512) || !applicationPlanID(s.TenantID, 64) || !applicationPlanID(s.AppID, 512) || !applicationPlanID(s.OwnerID, 512) || strings.TrimSpace(s.Title) == "" {
			return result, errors.New("MySQL session plan invalid session identity")
		}
		if seenSessions[s.ID] {
			return result, errors.New("MySQL session plan duplicate session id")
		}
		seenSessions[s.ID] = true
		tenants[s.TenantID] = true
		apps[s.AppID] = true
		result.Sessions++
		for _, msg := range s.Messages {
			if !applicationPlanID(msg.ID, 512) || msg.SessionID != s.ID {
				return result, errors.New("MySQL session plan invalid message identity")
			}
			if seenMessages[msg.ID] {
				return result, errors.New("MySQL session plan duplicate message id")
			}
			seenMessages[msg.ID] = true
			result.Messages++
		}
	}
	checksum := sha256.Sum256(root)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 4, 5
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.ApplySupported = true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	result.Tenants = len(tenants)
	result.Applications = len(apps)
	result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}
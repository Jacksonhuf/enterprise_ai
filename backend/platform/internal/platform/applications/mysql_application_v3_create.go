package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

// Internal persistence only. Caller must authorize the trusted tenant/owner.
// Blank config defaults remain a caller responsibility; template config comes
// from the locked source. No permissions, sessions or releases are inherited.
func createMySQLV3Application(ctx context.Context, db *sql.DB, tenant, owner string, config Config, template string, expected *string) (App, error) {
	var zero App
	if tenant == "" || owner == "" || strings.TrimSpace(config.Basic.Name) == "" || len([]rune(config.Basic.Name)) > 80 || (expected != nil && (template == "" || strings.TrimSpace(*expected) == "")) {
		return zero, errors.New("invalid relational create request")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
		return zero, err
	}
	if !mySQLRelationalRuntimeVersion(version) {
		return zero, errors.New("relational create requires schema 3 or 4")
	}
	st, _, err := readMySQLApplicationRelational(ctx, tx, false, version)
	if err != nil {
		return zero, err
	}
	var draft []byte
	if template != "" {
		source, ok := st.Apps[template]
		if !ok || source.TenantID != tenant {
			return zero, errRelationalAppUnavailable
		}
		if expected != nil && source.Revision != *expected {
			return zero, errRelationalDraftConflict
		}
		if err = tx.QueryRowContext(ctx, `SELECT raw_payload FROM cognara_fg_application_drafts WHERE tenant_id=? AND app_id=? FOR UPDATE`, tenant, template).Scan(&draft); err != nil {
			return zero, err
		}
		next := source.Draft
		next.Basic = config.Basic
		draft, err = mergeRelationalDraft(draft, next)
	} else {
		draft, err = json.Marshal(config)
	}
	if err != nil {
		return zero, err
	}
	var copied Config
	if err = json.Unmarshal(draft, &copied); err != nil {
		return zero, err
	}
	a := App{ID: ID("app"), AgentID: ID("agent"), TenantID: tenant, OwnerID: owner, Draft: copied, Revision: ID("draft"), Releases: []Release{}, UpdatedAt: Now()}
	raw, err := json.Marshal(a)
	if err != nil {
		return zero, err
	}
	var header map[string]json.RawMessage
	if err = json.Unmarshal(raw, &header); err != nil {
		return zero, err
	}
	delete(header, "draft")
	delete(header, "releases")
	raw, err = json.Marshal(header)
	if err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_applications(tenant_id,app_id,agent_id,owner_id,current_release_id,updated_at,metadata,raw_payload,releases_is_null) VALUES(?,?,?,?,NULL,?,?,?,0)`, tenant, a.ID, a.AgentID, owner, a.UpdatedAt, string(raw), raw); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_application_drafts(tenant_id,app_id,revision,config,raw_payload) VALUES(?,?,?,?,?)`, tenant, a.ID, a.Revision, string(draft), draft); err != nil {
		return zero, err
	}
	var last uint64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
		return zero, err
	}
	if err = insertMySQLAudit(ctx, tx, []Event{{ID: ID("event"), TenantID: tenant, ActorID: owner, OccurredAt: a.UpdatedAt, Action: "application.create", ResourceID: a.ID, Outcome: "success"}}, last); err != nil {
		return zero, err
	}
	if _, _, err = readMySQLApplicationRelational(ctx, tx, false, version); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return a, nil
}

package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyMySQLExceptionsCommitsSchema7(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema6ExceptionRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLExceptions(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || !plan.ApplySupported || plan.SourceSchemaVersion != 6 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLExceptions(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLExceptions(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after []byte
	if err = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); err != nil || version != 7 {
		t.Fatal("version not switched", version, err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(after, &object) != nil || object == nil {
		t.Fatal("invalid root after apply")
	}
	if _, ok := object["exceptions"]; ok {
		t.Fatal("exceptions still in root body")
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_exceptions`).Scan(&count); err != nil || count != plan.Exceptions {
		t.Fatal("exception rows missing", count, plan, err)
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected schema7", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Exceptions) != plan.Exceptions {
		t.Fatal("runtime state incomplete", err, len(st.Exceptions))
	}
	var before struct {
		Exceptions []map[string]any `json:"exceptions"`
	}
	if json.Unmarshal(root, &before) != nil {
		t.Fatal()
	}
	if len(before.Exceptions) != plan.Exceptions {
		t.Fatal("fixture mismatch", len(before.Exceptions))
	}
	byID := map[string]map[string]any{}
	for _, item := range before.Exceptions {
		id, _ := item["id"].(string)
		byID[id] = item
	}
	for _, got := range st.Exceptions {
		id, _ := got["id"].(string)
		want, ok := byID[id]
		if !ok {
			t.Fatal("missing exception after apply", id)
		}
		if want["status"] != got["status"] || want["revision"] != got["revision"] {
			t.Fatal("exception field diverged", id)
		}
	}
	record, handled, err := runtime.UpdateMySQLExceptionContext(ctx, "42", "exc-plan-1", func(item map[string]any) (*ExceptionPatch, error) {
		if item["revision"] != "r1" {
			t.Fatal("relational read mismatch")
		}
		return &ExceptionPatch{Status: "ACKNOWLEDGED", Revision: "r-applied", UpdatedAt: Now(), UpdatedBy: "demo-admin", Audit: []Event{{TenantID: "42", ID: "audit-1", OccurredAt: Now()}}}, nil
	})
	if err != nil || !handled || record == nil {
		t.Fatal("status update failed", err, handled)
	}
	_, _, handled, err = runtime.QueryMySQLExceptions(ctx, "42", "", 10, func(map[string]any) bool { return true }, func(m map[string]any) string { id, _ := m["id"].(string); return id })
	if err != nil || !handled {
		t.Fatal("governance query failed", err)
	}
	if _, err = ApplyMySQLExceptions(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema7(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema6ExceptionRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLExceptions(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLExceptions(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil {
		t.Fatal()
	}
	object["exceptions"] = json.RawMessage(`[]`)
	corrupt, _ := json.Marshal(object)
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("ambiguous schema7 root accepted")
	}
}
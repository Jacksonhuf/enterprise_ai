package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyMySQLUsageCommitsSchema8(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema7UsageRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLUsage(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || !plan.ApplySupported || plan.SourceSchemaVersion != 7 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLUsage(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLUsage(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after []byte
	if err = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); err != nil || version != 8 {
		t.Fatal("version not switched", version, err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(after, &object) != nil || object == nil {
		t.Fatal("invalid root after apply")
	}
	if _, ok := object["usage"]; ok {
		t.Fatal("usage still in root body")
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_usage`).Scan(&count); err != nil || count != plan.Usage {
		t.Fatal("usage rows missing", count, plan, err)
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected schema8", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Usage) != plan.Usage {
		t.Fatal("runtime state incomplete", err, len(st.Usage))
	}
	var before struct {
		Usage []map[string]any `json:"usage"`
	}
	if json.Unmarshal(root, &before) != nil {
		t.Fatal()
	}
	byID := map[string]map[string]any{}
	for _, item := range before.Usage {
		id, _ := item["event_id"].(string)
		byID[id] = item
	}
	for _, got := range st.Usage {
		id, _ := got["event_id"].(string)
		want, ok := byID[id]
		if !ok {
			t.Fatal("missing usage after apply", id)
		}
		wantRaw, _ := json.Marshal(want)
		gotRaw, _ := json.Marshal(got)
		if !equalApplicationJSON(wantRaw, gotRaw) {
			t.Fatal("usage field diverged", id)
		}
	}
	match := func(map[string]any) bool { return true }
	idOf := func(m map[string]any) string { id, _ := m["event_id"].(string); return id }
	_, _, handled, err := runtime.QueryMySQLUsage(ctx, "42", "", 10, match, idOf)
	if err != nil || !handled {
		t.Fatal("governance query failed", err, handled)
	}
	if err = runtime.UpdateOperationalContext(ctx, func(st *State) error {
		st.Usage = append(st.Usage, map[string]any{
			"event_id": "usage-append-8", "tenant_id": "42", "unit": "CALL", "capability": "metric", "count": 1, "occurred_at": Now(),
		})
		return nil
	}); err != nil {
		t.Fatal("append usage failed", err)
	}
	st, err = runtime.readSQLContext(ctx)
	if err != nil || len(st.Usage) != plan.Usage+1 {
		t.Fatal("relational append failed", len(st.Usage), err)
	}
	if _, err = ApplyMySQLUsage(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema8(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema7UsageRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLUsage(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLUsage(ctx, dsn, plan.SourceSHA256); err != nil {
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
	object["usage"] = json.RawMessage(`[]`)
	corrupt, _ := json.Marshal(object)
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("ambiguous schema8 root accepted")
	}
}
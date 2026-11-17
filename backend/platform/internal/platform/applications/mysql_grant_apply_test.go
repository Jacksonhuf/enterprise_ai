package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyMySQLGrantsCommitsSchema4(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema3GrantFixture(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLGrants(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || !plan.ApplySupported || plan.SourceSchemaVersion != 3 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLGrants(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLGrants(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after []byte
	if err = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); err != nil || version != 4 {
		t.Fatal("version not switched", version, err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(after, &object) != nil || object == nil {
		t.Fatal("invalid root after apply")
	}
	if _, ok := object["grants"]; ok {
		t.Fatal("grants still in root body")
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_grants`).Scan(&count); err != nil || count != plan.Grants {
		t.Fatal("grant rows missing", count, plan.Grants, err)
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected schema4", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Grants) != plan.Grants || len(st.Apps) < 1 {
		t.Fatal("runtime state incomplete", err, len(st.Grants), len(st.Apps))
	}
	var before map[string]json.RawMessage
	json.Unmarshal(root, &before)
	var expected []Grant
	if err = json.Unmarshal(before["grants"], &expected); err != nil {
		t.Fatal(err)
	}
	byID := map[string]Grant{}
	for _, g := range st.Grants {
		byID[g.ID] = g
	}
	for _, g := range expected {
		got, ok := byID[g.ID]
		if !ok {
			t.Fatal("missing grant after apply", g.ID)
		}
		beforeRaw, _ := json.Marshal(g)
		afterRaw, _ := json.Marshal(got)
		if !equalApplicationJSON(beforeRaw, afterRaw) {
			t.Fatal("grant payload diverged", g.ID)
		}
	}
	if _, err = ApplyMySQLGrants(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema4(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema3GrantFixture(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLGrants(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLGrants(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil {
		t.Fatal(err)
	}
	object["grants"] = json.RawMessage(`[]`)
	corrupt, _ := json.Marshal(object)
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("ambiguous schema4 root accepted")
	}
}
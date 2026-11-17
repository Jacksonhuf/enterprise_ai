package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyMySQLSessionsCommitsSchema5(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema4SessionRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLSessions(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || !plan.ApplySupported || plan.SourceSchemaVersion != 4 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLSessions(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLSessions(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after []byte
	if err = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); err != nil || version != 5 {
		t.Fatal("version not switched", version, err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(after, &object) != nil || object == nil {
		t.Fatal("invalid root after apply")
	}
	if _, ok := object["sessions"]; ok {
		t.Fatal("sessions still in root body")
	}
	var sessions, messages int
	if err = db.QueryRow(`SELECT (SELECT COUNT(*) FROM cognara_fg_sessions),(SELECT COUNT(*) FROM cognara_fg_session_messages)`).Scan(&sessions, &messages); err != nil || sessions != plan.Sessions || messages != plan.Messages {
		t.Fatal("session rows missing", sessions, messages, plan, err)
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected schema5", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Sessions) != plan.Sessions || len(st.Apps) < 1 || len(st.Grants) < 1 {
		t.Fatal("runtime state incomplete", err, len(st.Sessions), len(st.Grants))
	}
	var before map[string]json.RawMessage
	json.Unmarshal(root, &before)
	var expected map[string]Session
	if err = json.Unmarshal(before["sessions"], &expected); err != nil {
		t.Fatal(err)
	}
	for id, want := range expected {
		got, ok := st.Sessions[id]
		if !ok {
			t.Fatal("missing session after apply", id)
		}
		beforeRaw, _ := json.Marshal(want)
		afterRaw, _ := json.Marshal(got)
		if !equalApplicationJSON(beforeRaw, afterRaw) {
			t.Fatal("session payload diverged", id)
		}
	}
	if _, err = ApplyMySQLSessions(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema5(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema4SessionRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLSessions(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLSessions(ctx, dsn, plan.SourceSHA256); err != nil {
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
	object["sessions"] = json.RawMessage(`{}`)
	corrupt, _ := json.Marshal(object)
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("ambiguous schema5 root accepted")
	}
}

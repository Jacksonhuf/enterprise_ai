package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMySQLAuditSQLFiltersPreserveTimeAndOutcomeSemantics(t *testing.T) {
	f := setup(t)
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	f.store = store
	f.server.Store = store
	events := []struct{ stamp, outcome string }{
		{"2026-11-16T00:00:00.123456001Z", "success"}, {"2026-11-16t00:00:00.123456789z", "success"}, {"2026-11-16T08:00:00.123456789+08:00", "success"},
		{"2026-11-16T00:00:00.123457Z", "success"}, {"2026-11-16T00:00:00.123455999Z", "failed"},
		{"0001-01-01t00:00:00z", "success"}, {"1000-01-01T00:00:00+01:00", "success"}, {"9999-12-31T23:59:59-01:00", "success"},
		{"2026-11-16T00:00:00.123456789Z", "SUCCESS"}, {"2026-11-16T00:00:00.123456789Z", "success "},
	}
	if e = store.Update(func(st *app.State) error {
		for i, event := range events {
			st.Audit = append(st.Audit, app.Event{ID: fmt.Sprintf("filter-%02d", i), TenantID: "42", ActorID: "demo-user", Action: "filter", OccurredAt: event.stamp, Outcome: event.outcome})
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	query := func(outcome, from, to string) string {
		q := url.Values{"limit": {"1"}, "query": {"filter-"}}
		if outcome != "" {
			q.Set("outcome", outcome)
		}
		if from != "" {
			q.Set("from_time", from)
		}
		if to != "" {
			q.Set("to_time", to)
		}
		return q.Encode()
	}
	queries := []string{
		query("success", "2026-11-16T00:00:00.123456789Z", "2026-11-16T00:00:00.123456789Z"),
		query("success", "2026-11-16T00:00:00.123456002Z", "2026-11-16T00:00:00.123456788Z"),
		query("success", "2026-11-16T08:00:00.123456001+08:00", "2026-11-16T00:00:00.123457Z"),
		query("success", "0001-01-01T00:00:00Z", "0001-01-01T00:00:00Z"), query("success", "1000-01-01T00:00:00+01:00", "1000-01-01T00:00:00+01:00"),
		query("success", "9999-12-31T23:59:59-01:00", "9999-12-31T23:59:59-01:00"), query("success", "", ""), query("failed", "", ""),
	}
	// Compare all pages and cursor bytes against the existing v1 implementation.
	read := func(q string) []map[string]any {
		pages := []map[string]any{}
		cursor := ""
		for i := 0; i < 30; i++ {
			path := governanceBase + "audit?" + q
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			page := decode(t, f.request("demo-admin", "GET", path, nil, 200))
			delete(page, "sampled_at")
			pages = append(pages, page)
			next, ok := page["next_cursor"].(string)
			if !ok {
				return pages
			}
			cursor = next
		}
		t.Fatal("pagination stuck")
		return nil
	}
	before := map[string][]map[string]any{}
	for _, q := range queries {
		before[q] = read(q)
	}
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	for _, q := range queries {
		after := read(q)
		if !reflect.DeepEqual(before[q], after) {
			t.Fatalf("SQL filter changed semantics for %s: before=%v after=%v", q, before[q], after)
		}
	}
}

func TestMySQLImportedAuditCaseAliasesRemainQueryable(t *testing.T) {
	f := setup(t)
	dsn := testsupport.MySQLDSN(t)
	state := f.store.Read()
	state.Audit = nil
	stateRaw, _ := json.Marshal(state)
	rows := []json.RawMessage{
		json.RawMessage(`{"event_id":"alias-outcome","tenant_id":"42","actor_user_id":"demo-user","occurred_at":"2026-11-16T00:00:00Z","Outcome":"success"}`),
		json.RawMessage(`{"event_id":"alias-conflict","tenant_id":"42","actor_user_id":"demo-user","occurred_at":"2026-11-16T00:00:00Z","outcome":"failed","Outcome":"success"}`),
		json.RawMessage(`{"event_id":"alias-time","tenant_id":"42","actor_user_id":"demo-user","occurred_at":"2026-11-17T00:00:00Z","Occurred_At":"2026-11-16T00:00:00Z","outcome":"success"}`),
	}
	raw, _ := json.Marshal(map[string]any{"format": "cognara-fg-snapshot", "schema_version": 2, "state": json.RawMessage(stateRaw), "audit": rows})
	path := filepath.Join(t.TempDir(), "aliases.json")
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	if _, e := app.ImportMySQLSnapshot(context.Background(), dsn, path, hex.EncodeToString(sum[:]), 2); e != nil {
		t.Fatal(e)
	}
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	f.store = store
	f.server.Store = store
	// Baseline uses the actual stored payload's Go-decoded values, including aliases.
	baseline := setup(t)
	decoded := store.Read()
	if len(decoded.Audit) != 3 {
		t.Fatal("imported aliases unreadable")
	}
	if e = baseline.store.Update(func(st *app.State) error { *st = decoded; return nil }); e != nil {
		t.Fatal(e)
	}
	queries := []string{"outcome=success", "outcome=failed", "from_time=2026-11-17T00:00:00Z&to_time=2026-11-17T00:00:00Z", "outcome=success&from_time=2026-11-17T00:00:00Z&to_time=2026-11-17T00:00:00Z"}
	for _, query := range queries {
		path := governanceBase + "audit?" + query
		left := decode(t, baseline.request("demo-admin", "GET", path, nil, 200))
		right := decode(t, f.request("demo-admin", "GET", path, nil, 200))
		delete(left, "sampled_at")
		delete(right, "sampled_at")
		if !reflect.DeepEqual(left, right) {
			t.Fatal("case alias SQL mismatch", query, left, right)
		}
	}
}

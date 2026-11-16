package api_test

import (
	"context"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"net/url"
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

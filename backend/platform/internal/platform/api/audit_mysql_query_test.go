package api_test

import (
	"context"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"net/url"
	"reflect"
	"testing"
)

func TestMySQLAuditIndexedQueryMatchesV1AndPreservesPagination(t *testing.T) {
	f := setup(t)
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	f.store = store
	f.server.Store = store
	seedGovernanceAudit(t, f, 450)
	if e = store.Update(func(st *app.State) error {
		st.Audit = append(st.Audit, app.Event{ID: "foreign-tenant", TenantID: "43", ActorID: "demo-outsider", OccurredAt: app.Now()}, app.Event{ID: "legacy-own", ActorID: "demo-user", OccurredAt: "0001-01-01t00:00:00z", Outcome: "success"}, app.Event{ID: "legacy-foreign", ActorID: "demo-outsider", OccurredAt: app.Now()})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	queries := []string{"?limit=13", "?limit=13&query=audit-page-000", "?query=request-target&outcome=success&limit=7", "?query=" + url.QueryEscape("AUDIT-PAGE-44"), "?from_time=0001-01-01t00:00:00z&to_time=0001-01-01T00:00:00Z", "?query=foreign", "?query=legacy"}
	before := map[string]map[string]any{}
	for _, q := range queries {
		before[q] = decode(t, f.request("demo-admin", "GET", governanceBase+"audit"+q, nil, 200))
		delete(before[q], "sampled_at")
	}
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	for _, q := range queries {
		after := decode(t, f.request("demo-admin", "GET", governanceBase+"audit"+q, nil, 200))
		delete(after, "sampled_at")
		if !reflect.DeepEqual(before[q], after) {
			t.Fatalf("changed query semantics: %s\nbefore=%v\nafter=%v", q, before[q], after)
		}
	}
	// A v1 cursor remains valid after migration, and new events do not shift older pages.
	first := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?query=audit-page-&limit=17", nil, 200))
	seen := map[string]bool{}
	page := first
	pages := 0
	for {
		for _, item := range page["items"].([]any) {
			id := item.(map[string]any)["event_id"].(string)
			if seen[id] {
				t.Fatal("duplicate", id)
			}
			seen[id] = true
		}
		pages++
		if pages == 1 {
			if e = store.Update(func(st *app.State) error {
				st.Audit = append(st.Audit, app.Event{ID: "audit-page-new", TenantID: "42", ActorID: "demo-user", OccurredAt: app.Now()})
				return nil
			}); e != nil {
				t.Fatal(e)
			}
		}
		cursor, ok := page["next_cursor"].(string)
		if !ok {
			break
		}
		page = decode(t, f.request("demo-admin", "GET", governanceBase+"audit?query=audit-page-&limit=17&cursor="+url.QueryEscape(cursor), nil, 200))
	}
	if len(seen) != 450 || seen["audit-page-new"] {
		t.Fatal("append shifted history", len(seen))
	}
	old := before["?limit=13"]["next_cursor"].(string)
	f.request("demo-admin", "GET", governanceBase+"audit?limit=13&cursor="+url.QueryEscape(old), nil, 200)
	f.request("demo-admin", "GET", governanceBase+"audit?query=changed&cursor="+url.QueryEscape(old), nil, 400)
	f.request("demo-user", "GET", governanceBase+"audit", nil, 403)
	// Store-level tenant binding holds even without an application predicate.
	rows, _, handled, e := store.QueryMySQLAudit(context.Background(), "43", "", 100, func(event app.Event) bool { return event.TenantID == "43" })
	if e != nil || !handled || len(rows) != 1 || rows[0].ID != "foreign-tenant" {
		t.Fatal("tenant isolation", e, rows)
	}
}
func TestMySQLAuditIndexedQueryFindsSparseMatchesAcrossBatches(t *testing.T) {
	f := setup(t)
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	f.store = store
	f.server.Store = store
	seedGovernanceAudit(t, f, 430)
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	rows, more, handled, e := store.QueryMySQLAudit(context.Background(), "42", "", 1, func(event app.Event) bool { return event.ID == "audit-page-000" || event.ID == "audit-page-001" })
	if e != nil || !handled || !more || len(rows) != 1 || rows[0].ID != "audit-page-001" {
		t.Fatal("sparse query", e, rows)
	}
	rows, more, _, e = store.QueryMySQLAudit(context.Background(), "42", rows[0].ID, 1, func(event app.Event) bool { return event.ID == "audit-page-000" || event.ID == "audit-page-001" })
	if e != nil || more || len(rows) != 1 || rows[0].ID != "audit-page-000" {
		t.Fatal("sparse next", e, rows)
	}
}

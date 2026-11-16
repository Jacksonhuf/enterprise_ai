package api_test

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLExceptionQueryMatchesAllLegacyPages(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = store.Update(func(st *app.State) error {
		for i := 0; i < 450; i++ {
			status := "RESOLVED"
			if i%13 == 0 {
				status = "OPEN"
			}
			name := "库存异常"
			if i%37 == 0 {
				name = "sparse-hit"
			}
			stamp := "2026-11-16T00:00:00.123456789Z"
			if i%2 == 0 {
				stamp = "2026-11-16t00:00:00.123456001z"
			}
			if i == 7 {
				stamp = "bad"
			}
			if i == 8 {
				stamp = "0001-01-01T00:00:00Z"
			}
			item := map[string]any{"id": fmt.Sprintf("case-%03d", i), "tenant_id": "42", "name": name, "status": status, "revision": fmt.Sprint(i), "occurred_at": stamp}
			if i%101 == 0 {
				item["event_id"] = fmt.Sprintf("preferred-%03d", i)
			}
			st.Exceptions = append(st.Exceptions, item)
			if i%20 == 0 {
				st.Exceptions = append(st.Exceptions, map[string]any{"id": fmt.Sprintf("foreign-%03d", i), "tenant_id": "43", "name": "secret", "status": "OPEN"})
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	f := setup(t)
	f.store = store
	f.server.Store = store
	queries := []string{"limit=46", "limit=3&status=OPEN", "limit=1&query=sparse-hit", "limit=2&query=" + url.QueryEscape("库存异常") + "&status=OPEN", "limit=7&from_time=2026-11-16T00:00:00.123456789Z&to_time=2026-11-16T00:00:00.123456789Z", "limit=20&from_time=0001-01-01T00:00:00Z&to_time=0001-01-01T00:00:00Z", "limit=20&query=no-result"}
	read := func(query string) []map[string]any {
		pages := []map[string]any{}
		cursor := ""
		for i := 0; i < 100; i++ {
			path := governanceBase + "exceptions?" + query
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
		t.Fatal("pagination did not terminate")
		return nil
	}
	before := map[string][]map[string]any{}
	for _, query := range queries {
		before[query] = read(query)
	}
	// The existing v1 cursor must remain usable after switching to the v2 query path.
	oldCursor := before["limit=46"][0]["next_cursor"].(string)
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	oldPage := decode(t, f.request("demo-admin", "GET", governanceBase+"exceptions?limit=46&cursor="+url.QueryEscape(oldCursor), nil, 200))
	delete(oldPage, "sampled_at")
	if !reflect.DeepEqual(oldPage, before["limit=46"][1]) {
		t.Fatal("old cursor changed during migration")
	}
	for _, query := range queries {
		if after := read(query); !reflect.DeepEqual(before[query], after) {
			t.Fatal("SQL query changed legacy pages or cursors", query)
		}
	}
}
func TestMySQLExceptionQueryCursorAuthorizationAndStatusChanges(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = store.Update(func(st *app.State) error {
		for i := 0; i < 4; i++ {
			st.Exceptions = append(st.Exceptions, map[string]any{"id": fmt.Sprintf("item-%d", i), "tenant_id": "42", "status": "OPEN", "revision": "r1", "occurred_at": "2026-11-16T00:00:00Z"})
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	f := setup(t)
	f.store = store
	f.server.Store = store
	path := governanceBase + "exceptions?limit=1&status=OPEN"
	first := decode(t, f.request("demo-admin", "GET", path, nil, 200))
	cursor := first["next_cursor"].(string)
	if e = store.UpdateOperational(func(st *app.State) error {
		st.Exceptions = append(st.Exceptions, map[string]any{"id": "new-head", "tenant_id": "42", "status": "OPEN", "revision": "r1"})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	page := decode(t, f.request("demo-admin", "GET", path+"&cursor="+url.QueryEscape(cursor), nil, 200))
	if page["items"].([]any)[0].(map[string]any)["id"] != "item-2" {
		t.Fatal("append shifted old page")
	}
	f.request("demo-user", "GET", path, nil, 403)
	f.request("demo-outsider", "GET", path, nil, 403)
	f.request("demo-admin", "GET", governanceBase+"exceptions?limit=1&status=RESOLVED&cursor="+url.QueryEscape(cursor), nil, 400)
	f.request("demo-admin", "GET", governanceBase+"exceptions?cursor=bad", nil, 400)
	f.request("demo-admin", "GET", governanceBase+"exceptions?limit=0", nil, 400)
	f.request("demo-admin", "GET", governanceBase+"exceptions?outcome=failed", nil, 400)
	f.request("demo-admin", "POST", "/api/v1/cognitive/governance/exceptions/item-3/status", map[string]string{"expected_revision": "r1", "status": "RESOLVED"}, 200)
	f.request("demo-admin", "GET", path+"&cursor="+url.QueryEscape(cursor), nil, 400)
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	f.request("demo-admin", "GET", path, nil, 503)
}

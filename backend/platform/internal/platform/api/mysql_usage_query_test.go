package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLUsageQueryMatchesLegacyUnitsTimesAndAllPages(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = store.Update(func(st *app.State) error {
		for i := 0; i < 431; i++ {
			capability := "常规指标"
			if i%41 == 0 {
				capability = "sparse-hit"
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
			id := fmt.Sprintf("usage-%03d", i)
			item := map[string]any{"event_id": id, "tenant_id": "42", "capability": capability, "count": float64(i) / 10, "unit": []string{"CALL", "TOKEN", "MILLISECOND"}[i%3], "occurred_at": stamp}
			if i%91 == 0 {
				delete(item, "event_id")
				item["grant_id"] = "legacy-" + id
				item["id"] = "fallback-" + id
			}
			st.Usage = append(st.Usage, item)
			if i%19 == 0 {
				st.Usage = append(st.Usage, map[string]any{"event_id": fmt.Sprintf("foreign-%03d", i), "tenant_id": "43", "capability": "private", "count": 999, "unit": "TOKEN"})
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	f := setup(t)
	f.store = store
	f.server.Store = store
	queries := []string{"limit=67", "limit=17&unit=TOKEN", "limit=99&unit=MILLISECOND", "limit=1&query=sparse-hit", "limit=19&query=" + url.QueryEscape("常规指标") + "&unit=CALL", "limit=29&from_time=2026-11-16T00:00:00.123456789Z&to_time=2026-11-16T00:00:00.123456789Z", "limit=20&from_time=0001-01-01T00:00:00Z&to_time=0001-01-01T00:00:00Z", "limit=20&query=no-result"}
	read := func(query string) []map[string]any {
		pages := []map[string]any{}
		cursor := ""
		for i := 0; i < 100; i++ {
			path := governanceBase + "usage?" + query
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
	cursor := before["limit=67"][0]["next_cursor"].(string)
	raw, _ := base64.RawURLEncoding.DecodeString(cursor)
	var token map[string]any
	json.Unmarshal(raw, &token)
	if token["after"] != "legacy-usage-364" {
		t.Fatal("stable id fallback fixture did not cover cursor", token)
	}
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	old := decode(t, f.request("demo-admin", "GET", governanceBase+"usage?limit=67&cursor="+url.QueryEscape(cursor), nil, 200))
	delete(old, "sampled_at")
	if !reflect.DeepEqual(old, before["limit=67"][1]) {
		t.Fatal("old usage cursor changed")
	}
	for _, query := range queries {
		if after := read(query); !reflect.DeepEqual(after, before[query]) {
			t.Fatal("usage SQL query changed pages or cursor", query)
		}
	}
}
func TestMySQLUsageQueryDoesNotChangeReceiptsOrRedelivery(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = store.Update(func(st *app.State) error {
		for i := 0; i < 3; i++ {
			st.Usage = append(st.Usage, map[string]any{"event_id": fmt.Sprintf("old-%d", i), "tenant_id": "42", "capability": "metric", "count": 1, "unit": "CALL"})
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
	send := func(id string) string {
		body, _ := json.Marshal(map[string]any{"event_id": id, "occurred_at": "2026-11-16T00:00:00Z", "producer": "E", "tenant_id": "42", "capability": "metric", "count": 2, "unit": "CALL"})
		req, _ := http.NewRequest("POST", f.host.URL+"/internal/v1/usage-events", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+f.server.ServiceToken)
		req.Header.Set("Idempotency-Key", id)
		req.Header.Set("Content-Type", "application/json")
		response, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal("usage delivery", response.StatusCode, string(raw))
		}
		return decode(t, raw)["receipt_status"].(string)
	}
	if send("incoming") != "ACCEPTED" {
		t.Fatal("new event not accepted")
	}
	path := governanceBase + "usage?limit=1&unit=CALL"
	first := decode(t, f.request("demo-admin", "GET", path, nil, 200))
	cursor := first["next_cursor"].(string)
	if send("incoming") != "DUPLICATE" || send("new-head") != "ACCEPTED" {
		t.Fatal("receipt semantics changed")
	}
	page := decode(t, f.request("demo-admin", "GET", path+"&cursor="+url.QueryEscape(cursor), nil, 200))
	if page["items"].([]any)[0].(map[string]any)["event_id"] != "old-2" {
		t.Fatal("new usage shifted old page")
	}
	state := store.Read()
	if len(state.Usage) != 5 || len(state.Receipts) != 2 {
		t.Fatal("read/retry changed usage accounting")
	}
	f.request("demo-user", "GET", path, nil, 403)
	f.request("demo-outsider", "GET", path, nil, 403)
	f.request("demo-admin", "GET", governanceBase+"usage?limit=1&unit=TOKEN&cursor="+url.QueryEscape(cursor), nil, 400)
	f.request("demo-admin", "GET", governanceBase+"exceptions?cursor="+url.QueryEscape(cursor), nil, 400)
	f.request("demo-admin", "GET", governanceBase+"usage?status=OPEN", nil, 400)
	f.request("demo-admin", "GET", governanceBase+"usage?unit=call", nil, 400)
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.usage[3].unit','TOKEN')`); e != nil {
		t.Fatal(e)
	}
	f.request("demo-admin", "GET", path+"&cursor="+url.QueryEscape(cursor), nil, 400)
	if len(store.Read().Usage) != 5 || len(store.Read().Receipts) != 2 {
		t.Fatal("query errors changed accounting")
	}
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	f.request("demo-admin", "GET", path, nil, 503)
}

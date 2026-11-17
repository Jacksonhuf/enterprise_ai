package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLSettingsTargetedHTTPCompatibility(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			store, e := app.OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			if e = store.Update(func(st *app.State) error {
				st.Settings["demo-user:profile"] = json.RawMessage(`{"display_name":"旧名称"}`)
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			if version == 2 {
				if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
					t.Fatal(e)
				}
			}
			f := setup(t)
			f.store = store
			f.server.Store = store
			initial := settingsRead(t, f, "demo-user", "profile")
			first := settingsPut(t, f, "demo-user", "profile", initial["revision"].(string), map[string]string{"display_name": " 新名称 "}, 200)
			revision := first["revision"].(string)
			if first["values"].(map[string]any)["display_name"] != "新名称" {
				t.Fatal("name normalization changed")
			}
			settingsPut(t, f, "demo-user", "profile", initial["revision"].(string), map[string]string{"display_name": "stale"}, 409)
			noop := settingsPut(t, f, "demo-user", "profile", revision, map[string]string{"display_name": "新名称"}, 200)
			if noop["revision"] != revision || settingsAudit(store.Read()) != 1 {
				t.Fatal("no-op changed revision/audit")
			}
			if len(settingsRead(t, f, "demo-admin", "profile")["values"].(map[string]any)) != 0 || len(settingsRead(t, f, "demo-outsider", "profile")["values"].(map[string]any)) != 0 {
				t.Fatal("settings crossed identity")
			}
			f.request("demo-user", "PUT", settingsBase+"public", map[string]any{"values": map[string]string{"notice": "forbidden"}}, 403)
			general := settingsRead(t, f, "demo-user", "general")
			settingsPut(t, f, "demo-user", "general", general["revision"].(string), map[string]string{"density": "compact", "timezone": "Asia/Shanghai"}, 200)
			public := settingsRead(t, f, "demo-admin", "public")
			settingsPut(t, f, "demo-admin", "public", public["revision"].(string), map[string]string{"notice": "企业公告"}, 200)
			assertGovernanceNotice(t, f, "企业公告")
			settingsPut(t, f, "demo-user", "profile", revision, map[string]string{}, 200)
			me := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/me", nil, 200))
			if me["name"] != "张敏" {
				t.Fatal("clear resurrected legacy profile")
			}
			// A canonical alias forces the safe aggregate path rather than changing semantics.
			if version == 2 {
				db, e := sql.Open("mysql", dsn)
				if e != nil {
					t.Fatal(e)
				}
				defer db.Close()
				if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.SETTINGS',JSON_OBJECT())`); e != nil {
					t.Fatal(e)
				}
				current := settingsRead(t, f, "demo-user", "general")
				settingsPut(t, f, "demo-user", "general", current["revision"].(string), map[string]string{"density": "comfortable"}, 200)
			}
		})
	}
}
func TestMySQLSettingsTargetedHTTPConcurrentCASPreservesOtherDomains(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	a, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	first, second := setup(t), setup(t)
	first.store = a
	first.server.Store = a
	second.store = b
	second.server.Store = b
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.apps."channel-replenishment".future',CAST(? AS JSON))`, `{"exact":9007199254740993}`); e != nil {
		t.Fatal(e)
	}
	revision := settingsRead(t, first, "demo-user", "general")["revision"].(string)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	errs := make(chan error, 2)
	for i, f := range []*fixture{first, second} {
		wg.Add(1)
		go func(i int, f *fixture) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{"expected_revision": revision, "values": map[string]string{"density": []string{"compact", "comfortable"}[i]}})
			req, _ := http.NewRequest("PUT", f.host.URL+settingsBase+"general", bytes.NewReader(raw))
			req.Header.Set("Authorization", "Bearer demo-user")
			req.Header.Set("Content-Type", "application/json")
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				errs <- e
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			results <- resp.StatusCode
		}(i, f)
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	counts := map[int]int{}
	for status := range results {
		counts[status]++
	}
	if counts[200] != 1 || counts[409] != 1 || settingsAudit(a.Read()) != 1 {
		t.Fatal("CAS lost", counts)
	}
	var future string
	if e = db.QueryRow(`SELECT JSON_EXTRACT(body,'$.apps."channel-replenishment".future') FROM cognara_fg_state WHERE id=1`).Scan(&future); e != nil || !strings.Contains(future, "9007199254740993") {
		t.Fatal("targeted write rewrote another domain", e)
	}
}

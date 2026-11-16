package api_test

import (
	"bytes"
	"encoding/json"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostgresConcurrentSessionRenames(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	f := setup(t)
	a, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	f.store = a
	f.server.Store = a
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	id := f.session()
	rev := sessionMeta(t, f, id)["revision"].(string)
	raw, _ := json.Marshal(map[string]string{"title": "并发新标题", "expected_revision": rev})
	type result struct {
		status int
		err    error
	}
	results := make(chan result, 2)
	for _, address := range []string{f.host.URL, host.URL} {
		go func(address string) {
			r, _ := http.NewRequest("PUT", address+"/api/v1/sessions/"+id, bytes.NewReader(raw))
			r.Header.Set("Authorization", "Bearer demo-user")
			r.Header.Set("Content-Type", "application/json")
			response, e := http.DefaultClient.Do(r)
			if e != nil {
				results <- result{err: e}
				return
			}
			response.Body.Close()
			results <- result{status: response.StatusCode}
		}(address)
	}
	statuses := map[int]int{}
	for i := 0; i < 2; i++ {
		v := <-results
		if v.err != nil {
			t.Fatal(v.err)
		}
		statuses[v.status]++
	}
	if statuses[200] != 1 || statuses[409] != 1 {
		t.Fatal(statuses)
	}
	reopened, e := app.OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	v := reopened.Read().Sessions[id]
	count := 0
	for _, event := range reopened.Read().Audit {
		if event.Action == "session.rename" && event.ResourceID == id {
			count++
			if event.RequestID == "" {
				t.Fatal("missing request attribution")
			}
		}
	}
	if v.Title != "并发新标题" || v.Revision == rev || count != 1 {
		t.Fatal("rename/audit not atomic")
	}
	// Authorization is rechecked even when the client still has a valid metadata revision.
	if e = a.Update(func(st *app.State) error {
		for key, g := range st.Grants {
			if g.ResourceID == v.AppID && g.Action == "application.use" {
				g.Active = false
				st.Grants[key] = g
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	rename(t, f, id, v.Revision, "撤权后修改", 403)
	f.request("demo-user", "GET", "/api/v1/sessions?app_id="+v.AppID, nil, 403)
	if a.Read().Sessions[id].Title != v.Title {
		t.Fatal("revoked rename changed title")
	}
}

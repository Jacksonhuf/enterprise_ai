package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const grantPath = "/api/v1/cognitive/governance/grants"

func grantPayload(subjectType, subjectID, resourceType, resourceID, action string) map[string]string {
	return map[string]string{"subject_type": subjectType, "subject_id": subjectID, "resource_type": resourceType, "resource_id": resourceID, "action": action, "scope": "east"}
}
func countGrantAudit(st app.State, action string) int {
	n := 0
	for _, e := range st.Audit {
		if e.Action == action {
			n++
		}
	}
	return n
}
func TestGrantRejectsForeignSubjectsAndResources(t *testing.T) {
	f := setup(t)
	if err := f.store.Update(func(st *app.State) error {
		foreign := st.Apps["channel-replenishment"]
		foreign.ID = "foreign-app"
		foreign.TenantID = "43"
		st.Apps[foreign.ID] = foreign
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := len(f.store.Read().Grants)
	cases := []map[string]string{
		grantPayload("APPLICATION", "foreign-app", "SERVICE", "replenishment-gap", "service.invoke"),
		grantPayload("USER", "demo-outsider", "APPLICATION", "channel-replenishment", "application.use"),
		grantPayload("USER", "demo-user", "APPLICATION", "foreign-app", "application.use"),
		grantPayload("APPLICATION", "missing-app", "ASSET", "chip-gap-metric", "execution.invoke"),
	}
	for _, body := range cases {
		f.request("demo-admin", "POST", grantPath, body, 400)
	}
	if len(f.store.Read().Grants) != before || countGrantAudit(f.store.Read(), "authorization.grant") != 0 {
		t.Fatal("rejected grant persisted")
	}
	f.request("demo-user", "POST", grantPath, grantPayload("USER", "demo-user", "APPLICATION", "channel-replenishment", "application.use"), 403)
}
func TestExistingActiveGrantReusedWithoutAudit(t *testing.T) {
	f := setup(t)
	before := len(f.store.Read().Grants)
	body := grantPayload("APPLICATION", "channel-replenishment", "ASSET", "chip-gap-metric", "asset.context.read")
	for i := 0; i < 3; i++ {
		response := decode(t, f.request("demo-admin", "POST", grantPath, body, 200))
		if response["grant_id"] != "grant-metric-context" {
			t.Fatal("existing grant replaced")
		}
	}
	if len(f.store.Read().Grants) != before || countGrantAudit(f.store.Read(), "authorization.grant") != 0 {
		t.Fatal("no-op grant created rows or audit")
	}
}
func assertConcurrentGrant(t *testing.T, f *fixture, hosts []string) {
	t.Helper()
	f.request("demo-admin", "DELETE", grantPath+"/grant-app", nil, 200)
	body, _ := json.Marshal(grantPayload("USER", "demo-user", "APPLICATION", "channel-replenishment", "application.use"))
	before := len(f.store.Read().Grants)
	type outcome struct {
		id  string
		err error
	}
	outcomes := make(chan outcome, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			r, _ := http.NewRequest("POST", host+grantPath, bytes.NewReader(body))
			r.Header.Set("Authorization", "Bearer demo-admin")
			r.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(r)
			if err != nil {
				outcomes <- outcome{err: err}
				return
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)
			var grant app.Grant
			err = json.Unmarshal(raw, &grant)
			if resp.StatusCode != 200 || grant.ID == "" {
				err = fmt.Errorf("grant response status %d", resp.StatusCode)
			}
			outcomes <- outcome{grant.ID, err}
		}(hosts[i%len(hosts)])
	}
	wg.Wait()
	close(outcomes)
	id := ""
	for o := range outcomes {
		if o.err != nil {
			t.Fatal(o.err)
		}
		if id == "" {
			id = o.id
		}
		if id != o.id {
			t.Fatal("concurrent duplicate grants")
		}
	}
	state := f.store.Read()
	if len(state.Grants) != before+1 || countGrantAudit(state, "authorization.grant") != 1 {
		t.Fatal("duplicate rows or audit", len(state.Grants))
	}
	revokeBefore := countGrantAudit(state, "authorization.revoke")
	for i := 0; i < 2; i++ {
		f.request("demo-admin", "DELETE", grantPath+"/"+id, nil, 200)
	}
	if countGrantAudit(f.store.Read(), "authorization.revoke") != revokeBefore+1 {
		t.Fatal("repeated revocation duplicated audit")
	}
	catalog := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/apps?purpose=USE", nil, 200))
	if len(catalog["items"].([]any)) != 0 {
		t.Fatal("revoked access remained")
	}
	renewed := decode(t, f.request("demo-admin", "POST", grantPath, grantPayload("USER", "demo-user", "APPLICATION", "channel-replenishment", "application.use"), 200))
	if renewed["grant_id"] == id {
		t.Fatal("revoked row silently reactivated")
	}
	if countGrantAudit(f.store.Read(), "authorization.grant") != 2 {
		t.Fatal("regrant not audited")
	}
}
func TestConcurrentGrantsAndRevocationIdempotency(t *testing.T) {
	f := setup(t)
	assertConcurrentGrant(t, f, []string{f.host.URL})
}
func TestPostgresConcurrentGrantsAcrossHosts(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	f := setup(t)
	a, err := app.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := app.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	f.store = a
	f.server.Store = a
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	defer host.Close()
	assertConcurrentGrant(t, f, []string{f.host.URL, host.URL})
	if len(b.Read().Grants) != len(a.Read().Grants) {
		t.Fatal("second connection did not observe regrant")
	}
}

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

func draftSaves(st app.State) int {
	n := 0
	for _, e := range st.Audit {
		if e.Action == "application.draft.save" {
			n++
		}
	}
	return n
}
func TestDraftSaveAuditAndConflictPreservePublishedSnapshot(t *testing.T) {
	f := setup(t)
	before := f.store.Read().Apps["channel-replenishment"]
	config := before.Draft
	config.Basic.Name = "Saved draft"
	body := struct {
		Expected string `json:"expected_draft_revision"`
		app.Config
	}{before.Revision, config}
	saved := decode(t, f.request("demo-engineer", "PUT", "/api/v1/cognitive/apps/"+before.ID, body, 200))
	if saved["draft_revision"] == before.Revision || draftSaves(f.store.Read()) != 1 {
		t.Fatal("save version/audit missing")
	}
	f.request("demo-engineer", "PUT", "/api/v1/cognitive/apps/"+before.ID, body, 409)
	after := f.store.Read().Apps[before.ID]
	current, _ := app.Current(after)
	original, _ := app.Current(before)
	if after.Draft.Basic.Name != "Saved draft" || after.Revision != saved["draft_revision"] || current.ID != original.ID || current.Config.Basic.Name != original.Config.Basic.Name || draftSaves(f.store.Read()) != 1 {
		t.Fatal("conflict changed draft/release/audit")
	}
	for _, e := range f.store.Read().Audit {
		if e.Action == "application.draft.save" && (e.RequestID == "" || e.ActorID != "demo-engineer" || e.ResourceID != before.ID) {
			t.Fatal("draft audit lost request attribution")
		}
	}
}
func TestPostgresConcurrentDraftSavesHaveOneWinner(t *testing.T) {
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
	before := a.Read().Apps["channel-replenishment"]
	config := before.Draft
	config.Basic.Name = "Concurrent draft"
	raw, _ := json.Marshal(struct {
		Expected string `json:"expected_draft_revision"`
		app.Config
	}{before.Revision, config})
	type result struct {
		status   int
		revision string
		err      error
	}
	results := make(chan result, 2)
	for _, address := range []string{f.host.URL, host.URL} {
		go func(address string) {
			r, _ := http.NewRequest("PUT", address+"/api/v1/cognitive/apps/"+before.ID, bytes.NewReader(raw))
			r.Header.Set("Authorization", "Bearer demo-engineer")
			r.Header.Set("Content-Type", "application/json")
			response, e := http.DefaultClient.Do(r)
			if e != nil {
				results <- result{err: e}
				return
			}
			defer response.Body.Close()
			var payload struct {
				Revision string `json:"draft_revision"`
			}
			e = json.NewDecoder(response.Body).Decode(&payload)
			results <- result{response.StatusCode, payload.Revision, e}
		}(address)
	}
	successes, conflicts := 0, 0
	revision := ""
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		switch r.status {
		case 200:
			successes++
			revision = r.revision
		case 409:
			conflicts++
		default:
			t.Fatal("unexpected status", r.status)
		}
	}
	if successes != 1 || conflicts != 1 || revision == "" || draftSaves(a.Read()) != 1 {
		t.Fatal("concurrent save/audit duplicated")
	}
	reopened, err := app.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after := reopened.Read().Apps[before.ID]
	published, _ := app.Current(after)
	original, _ := app.Current(before)
	if after.Revision != revision || after.Draft.Basic.Name != "Concurrent draft" || published.ID != original.ID || published.Config.Basic.Name != original.Config.Basic.Name || draftSaves(reopened.Read()) != 1 {
		t.Fatal("durable draft/published snapshot changed")
	}
}

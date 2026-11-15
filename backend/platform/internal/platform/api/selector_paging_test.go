package api_test

import (
	"encoding/json"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAssetSelectorPaginatesAuthorizedResults(t *testing.T) {
	f := setup(t)
	first := decode(t, f.request("demo-engineer", "GET", "/api/v1/cognitive/selectors/assets?limit=1", nil, 200))
	if len(first["items"].([]any)) != 1 || first["next_cursor"] != "1" {
		t.Fatal("first page not bounded", first)
	}
	second := decode(t, f.request("demo-engineer", "GET", "/api/v1/cognitive/selectors/assets?limit=1&cursor=1", nil, 200))
	if len(second["items"].([]any)) != 1 || second["next_cursor"] != nil {
		t.Fatal("second page missing", second)
	}
	a := first["items"].([]any)[0].(map[string]any)["asset_id"]
	b := second["items"].([]any)[0].(map[string]any)["asset_id"]
	if a == b {
		t.Fatal("duplicate page")
	}
	f.request("demo-engineer", "GET", "/api/v1/cognitive/selectors/assets?cursor=bad", nil, 400)
	f.request("demo-user", "GET", "/api/v1/cognitive/selectors/assets?limit=1", nil, 403)
}
func TestServiceSelectorForwardsOpaqueCursorAndRejectsInvalidBounds(t *testing.T) {
	f := setup(t)
	calls := make(chan map[string]any, 8)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		calls <- body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[],"next_cursor":null}`))
	}))
	defer upstream.Close()
	f.client.BaseURL = upstream.URL
	f.request("demo-engineer", "GET", "/api/v1/cognitive/selectors/services?limit=7&cursor="+url.QueryEscape("opaque:page/2")+"&query="+url.QueryEscape("补货"), nil, 200)
	body := <-calls
	if body["limit"] != float64(7) || body["cursor"] != "opaque:page/2" || body["query"] != "补货" || body["channel"] != "INTERNAL_APPLICATION" {
		t.Fatal("discovery parameters lost", body)
	}
	for _, kind := range []string{"assets", "services"} {
		for _, query := range []string{"limit=0", "limit=101", "limit=no", "query=" + url.QueryEscape(strings.Repeat("汉", 257)), "cursor=" + strings.Repeat("x", 2049)} {
			f.request("demo-engineer", "GET", "/api/v1/cognitive/selectors/"+kind+"?"+query, nil, 400)
		}
	}
	if len(calls) != 0 {
		t.Fatal("invalid request reached upstream")
	}
}
func TestDraftRejectsAmbiguousServiceVersions(t *testing.T) {
	f := setup(t)
	before := f.store.Read().Apps["channel-replenishment"]
	for _, version := range []string{"v1.1", "v2"} {
		config := before.Draft
		config.CapabilityRefs.ServiceRefs = []app.ServiceRef{{ServiceID: "replenishment-gap", Version: "v1.1"}, {ServiceID: "replenishment-gap", Version: version}}
		raw, _ := json.Marshal(config)
		body := map[string]any{}
		json.Unmarshal(raw, &body)
		body["expected_draft_revision"] = before.Revision
		f.request("demo-engineer", "PUT", "/api/v1/cognitive/apps/channel-replenishment", body, 400)
	}
	after := f.store.Read().Apps[before.ID]
	if after.Revision != before.Revision || len(after.Draft.CapabilityRefs.ServiceRefs) != 1 {
		t.Fatal("rejected configuration changed draft")
	}
	if err := f.store.Update(func(st *app.State) error {
		a := st.Apps[before.ID]
		a.Draft.CapabilityRefs.ServiceRefs = append(a.Draft.CapabilityRefs.ServiceRefs, a.Draft.CapabilityRefs.ServiceRefs[0])
		st.Apps[a.ID] = a
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/channel-replenishment/release-checks", map[string]string{"expected_draft_revision": before.Revision}, 400)
}

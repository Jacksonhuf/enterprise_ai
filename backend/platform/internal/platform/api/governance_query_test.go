package api_test

import (
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/url"
	"strings"
	"testing"
)

const governanceBase = "/api/v1/cognitive/governance/"

func seedGovernanceAudit(t *testing.T, f *fixture, n int) {
	t.Helper()
	if err := f.store.Update(func(st *app.State) error {
		for i := 0; i < n; i++ {
			st.Audit = append(st.Audit, app.Event{ID: fmt.Sprintf("audit-page-%03d", i), TenantID: "42", ActorID: "demo-user", ClientID: "app-search", Action: "service.call", ResourceID: "inventory", RequestID: "request-target", ExecutionID: "exec-target", OccurredAt: "2026-11-16T00:00:00Z", Outcome: "success"})
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestGovernanceSearchCorrelationAndInclusiveTime(t *testing.T) {
	f := setup(t)
	seedGovernanceAudit(t, f, 3)
	if err := f.store.Update(func(st *app.State) error {
		st.Audit = append(st.Audit, app.Event{ID: "foreign-audit", TenantID: "43", ActorID: "demo-outsider", Action: "hidden", RequestID: "request-target", OccurredAt: app.Now(), Outcome: "success"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"request-target", "EXEC-TARGET", "app-search", "demo-user"} {
		result := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?query="+url.QueryEscape(query)+"&outcome=success&from_time="+url.QueryEscape("2026-11-16T08:00:00+08:00")+"&to_time=2026-11-16T00:00:00Z", nil, 200))
		if len(result["items"].([]any)) != 3 {
			t.Fatal("correlation search or instant boundary incorrect", result)
		}
	}
	empty := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?query=foreign-audit", nil, 200))
	if len(empty["items"].([]any)) != 0 {
		t.Fatal("foreign audit leaked")
	}
	ancient := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?to_time=0001-01-01T00:00:00Z", nil, 200))
	if len(ancient["items"].([]any)) != 0 {
		t.Fatal("zero-time upper bound ignored")
	}
}
func TestGovernanceCursorSurvivesNewAuditAppend(t *testing.T) {
	f := setup(t)
	seedGovernanceAudit(t, f, 45)
	cursor := ""
	seen := map[string]bool{}
	page := 0
	for {
		result := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?limit=7&query=audit-page-&cursor="+url.QueryEscape(cursor), nil, 200))
		for _, raw := range result["items"].([]any) {
			id := raw.(map[string]any)["event_id"].(string)
			if seen[id] {
				t.Fatal("duplicate row", id)
			}
			seen[id] = true
		}
		page++
		if page == 1 {
			if err := f.store.Update(func(st *app.State) error {
				st.Audit = append(st.Audit, app.Event{ID: "audit-page-new", TenantID: "42", ActorID: "demo-user", Action: "call", OccurredAt: app.Now(), Outcome: "success"})
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		next, ok := result["next_cursor"].(string)
		if !ok {
			break
		}
		if next == cursor {
			t.Fatal("cursor stuck")
		}
		cursor = next
	}
	if len(seen) != 45 || seen["audit-page-new"] {
		t.Fatal("append shifted pagination", len(seen))
	}
}
func TestGovernanceFiltersAndCursorsRejectInvalidRequests(t *testing.T) {
	f := setup(t)
	seedGovernanceAudit(t, f, 3)
	first := decode(t, f.request("demo-admin", "GET", governanceBase+"audit?limit=1", nil, 200))
	cursor := url.QueryEscape(first["next_cursor"].(string))
	for _, query := range []string{"limit=0", "limit=101", "limit=no", "cursor=bad", "outcome=UNKNOWN", "status=OPEN", "unit=CALL", "from_time=bad", "from_time=2026-11-16T00:00:01Z&to_time=2026-11-16T00:00:00Z", "query=" + strings.Repeat("x", 257), "cursor=" + cursor + "&query=changed"} {
		f.request("demo-admin", "GET", governanceBase+"audit?"+query, nil, 400)
	}
	f.request("demo-admin", "GET", governanceBase+"usage?cursor="+cursor, nil, 400)
	f.request("demo-user", "GET", governanceBase+"audit?limit=1", nil, 403)
	f.request("demo-outsider", "GET", governanceBase+"audit?cursor="+cursor, nil, 403)
}
func TestGovernanceStateFiltersAndChangedCursor(t *testing.T) {
	f := setup(t)
	first := decode(t, f.request("demo-admin", "GET", governanceBase+"grants?limit=1&status=ACTIVE", nil, 200))
	row := first["items"].([]any)[0].(map[string]any)
	cursor := first["next_cursor"].(string)
	f.request("demo-admin", "DELETE", grantPath+"/"+row["grant_id"].(string), nil, 200)
	f.request("demo-admin", "GET", governanceBase+"grants?status=ACTIVE&cursor="+url.QueryEscape(cursor), nil, 400)
	revoked := decode(t, f.request("demo-admin", "GET", governanceBase+"grants?status=REVOKED", nil, 200))
	if len(revoked["items"].([]any)) != 1 {
		t.Fatal("revoked filter")
	}
	if err := f.store.Update(func(st *app.State) error {
		st.Exceptions = append(st.Exceptions, map[string]any{"id": "open-local", "tenant_id": "42", "name": "local", "status": "OPEN", "occurred_at": "2026-11-16T00:00:00Z"}, map[string]any{"id": "resolved-local", "tenant_id": "42", "status": "RESOLVED", "occurred_at": "2026-11-16T00:00:00Z"}, map[string]any{"id": "foreign-exception", "tenant_id": "43", "status": "OPEN", "occurred_at": "2026-11-16T00:00:00Z"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	open := decode(t, f.request("demo-admin", "GET", governanceBase+"exceptions?status=OPEN", nil, 200))
	if len(open["items"].([]any)) != 1 || open["items"].([]any)[0].(map[string]any)["id"] != "open-local" {
		t.Fatal("exception status/tenant filter")
	}
	f.request("demo-admin", "GET", governanceBase+"grants?from_time=2026-11-16T00:00:00Z", nil, 400)
}
func TestGovernanceUsageUnitAndTenantPagination(t *testing.T) {
	f := setup(t)
	if err := f.store.Update(func(st *app.State) error {
		for i := 0; i < 30; i++ {
			unit := "CALL"
			if i%2 == 0 {
				unit = "TOKEN"
			}
			st.Usage = append(st.Usage, map[string]any{"event_id": fmt.Sprintf("usage-page-%02d", i), "tenant_id": "42", "capability": "test", "unit": unit, "count": 1, "occurred_at": "2026-11-16T00:00:00Z"})
		}
		st.Usage = append(st.Usage, map[string]any{"event_id": "foreign-usage", "tenant_id": "43", "unit": "TOKEN", "count": 999, "occurred_at": "2026-11-16T00:00:00Z"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cursor := ""
	count := 0
	for {
		result := decode(t, f.request("demo-admin", "GET", governanceBase+"usage?unit=TOKEN&limit=7&cursor="+url.QueryEscape(cursor), nil, 200))
		for _, raw := range result["items"].([]any) {
			row := raw.(map[string]any)
			if row["unit"] != "TOKEN" || row["tenant_id"] != "42" || row["count"] != float64(1) {
				t.Fatal("mixed usage or foreign tenant")
			}
			count++
		}
		next, ok := result["next_cursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if count != 15 {
		t.Fatal("usage truncated", count)
	}
}

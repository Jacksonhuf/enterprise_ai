package api_test

import (
 "context"
 "encoding/json"
 "net/url"
 "reflect"
 "testing"
 app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func createTraceFailure(t *testing.T, f *fixture) (string, map[string]any) {
 t.Helper()
 envelope := decode(t, f.request("demo-admin", "POST", "/api/v1/cognitive/models/not-configured/test", map[string]any{}, 503))
 request, ok := envelope["request_id"].(string)
 if !ok || request == "" { t.Fatal("failure must expose a request identifier", envelope) }
 for _, item := range f.store.Read().Exceptions {
  if item["request_id"] == request { return request, item }
 }
 t.Fatal("failure was not recorded as an exception")
 return "", nil
}
func TestExceptionRequestTraceAndStateLifecycleDoNotReplayBusinessExecution(t *testing.T) {
 f := setup(t)
 request, original := createTraceFailure(t, f)
 trace := "/api/v1/cognitive/governance/audit?query="+url.QueryEscape(request)
 listing := decode(t, f.request("demo-admin", "GET", trace, nil, 200))
 events := listing["items"].([]any)
 if len(events) == 0 { t.Fatal("origin audit missing") }
 for _, item := range events {
  event := item.(map[string]any)
  if event["request_id"] != request || event["outcome"] != "failed" { t.Fatal("unrelated audit was returned", event) }
 }
 before := f.store.Read()
 modelTests := countGrantAudit(before, "model.connection.test")
 path := "/api/v1/cognitive/governance/exceptions/"+original["id"].(string)+"/status"
 update := func(revision any, status string) map[string]any {
  return decode(t, f.request("demo-admin", "POST", path, map[string]any{"expected_revision":revision,"status":status}, 200))
 }
 acknowledged := update(original["revision"], "ACKNOWLEDGED")
 f.request("demo-admin", "POST", path, map[string]any{"expected_revision":original["revision"],"status":"RESOLVED"}, 409)
 resolved := update(acknowledged["revision"], "RESOLVED")
 reopened := update(resolved["revision"], "OPEN")
 if acknowledged["status"] != "ACKNOWLEDGED" || resolved["status"] != "RESOLVED" || reopened["status"] != "OPEN" { t.Fatal("state lifecycle mismatch") }
 if reopened["request_id"] != request || reopened["revision"] == original["revision"] { t.Fatal("correlation or revision lost") }
 after := f.store.Read()
 if len(after.Exceptions) != len(before.Exceptions) || !reflect.DeepEqual(after.Usage,before.Usage) || !reflect.DeepEqual(after.Sessions,before.Sessions) || countGrantAudit(after,"model.connection.test") != modelTests { t.Fatal("status changes replayed execution or created failures") }
 f.request("demo-user", "GET", trace, nil, 403)
 f.request("demo-user", "POST", path, map[string]any{"expected_revision":reopened["revision"],"status":"ACKNOWLEDGED"}, 403)
}
func TestRequestTraceKeepsTenantIsolationEvenWhenRequestIdentifiersCoincide(t *testing.T) {
 f := setup(t)
 request, original := createTraceFailure(t, f)
 raw, _ := json.Marshal(original)
 foreign := map[string]any{}
 json.Unmarshal(raw, &foreign)
 foreign["id"] = "foreign-exception"
 foreign["tenant_id"] = "43"
 if err := f.store.UpdateOperationalContext(context.Background(), func(st *app.State) error {
  st.Exceptions = append(st.Exceptions, foreign)
  st.Audit = append(st.Audit, app.Event{TenantID:"43",ID:"event-foreign",OccurredAt:app.Now(),ActorID:"demo-outsider",Action:"foreign.audit",Outcome:"failed",RequestID:request})
  return nil
 }); err != nil { t.Fatal(err) }
 for _, section := range []string{"audit","exceptions"} {
  listing := decode(t, f.request("demo-admin", "GET", "/api/v1/cognitive/governance/"+section+"?query="+url.QueryEscape(request), nil, 200))
  items := listing["items"].([]any)
  if len(items) == 0 || section == "exceptions" && len(items) != 1 { t.Fatal("request filtering returned invalid records",section,listing) }
  for _, item := range items { if item.(map[string]any)["tenant_id"] != "42" { t.Fatal("foreign tenant appeared in trace") } }
 }
 f.request("demo-admin", "POST", "/api/v1/cognitive/governance/exceptions/foreign-exception/status", map[string]any{"expected_revision":foreign["revision"],"status":"ACKNOWLEDGED"}, 404)
}

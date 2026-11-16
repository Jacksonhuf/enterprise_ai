package api_test

import (
 "encoding/json"
 "strings"
 "testing"
)

func publishGrantFlowCopy(t *testing.T, f *fixture) string {
 t.Helper()
 source := f.store.Read().Apps["channel-replenishment"]
 copied := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps", map[string]any{
  "creation_mode":"TEMPLATE", "name":"授权流程副本", "template_id":source.ID, "expected_template_revision":source.Revision,
 }, 200))
 id := copied["app_id"].(string)
 revision := copied["draft_revision"]
 check := decode(t, f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+id+"/release-checks", map[string]any{"expected_draft_revision":revision}, 200))
 if check["status"] != "PASS" { t.Fatal(check) }
 f.request("demo-engineer", "POST", "/api/v1/cognitive/apps/"+id+"/releases", map[string]any{
  "check_id":check["check_id"], "expected_draft_revision":revision, "expected_current_release_id":nil, "release_note":"授权流程验证",
 }, 200)
 return id
}
func grantFlowVisible(t *testing.T, f *fixture, id string) bool {
 t.Helper()
 catalog := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/apps?purpose=USE", nil, 200))
 for _, row := range catalog["items"].([]any) {
  if row.(map[string]any)["app_id"] == id { return true }
 }
 return false
}
func TestPublishedCopyVisibilityAndSessionAccessFollowUserGrantLifecycle(t *testing.T) {
 f := setup(t)
 id := publishGrantFlowCopy(t, f)
 sessionPath := "/api/v1/cognitive/apps/"+id+"/sessions"
 if grantFlowVisible(t, f, id) { t.Fatal("publishing implicitly granted user access") }
 f.request("demo-user", "POST", sessionPath, map[string]any{}, 403)
 before := countGrantAudit(f.store.Read(), "authorization.grant")
 body := grantPayload("USER", "demo-user", "APPLICATION", id, "application.use")
 grant := decode(t, f.request("demo-admin", "POST", grantPath, body, 200))
 duplicate := decode(t, f.request("demo-admin", "POST", grantPath, body, 200))
 if duplicate["grant_id"] != grant["grant_id"] || countGrantAudit(f.store.Read(), "authorization.grant") != before+1 { t.Fatal("grant replay duplicated mutation") }
 if !grantFlowVisible(t, f, id) { t.Fatal("grant did not expose published copy") }
 session := f.request("demo-user", "POST", sessionPath, map[string]any{}, 200)
 var metadata map[string]any
 if err := json.Unmarshal(session, &metadata); err != nil || metadata["session_id"] == nil { t.Fatal("authorized session not created", string(session)) }
 grantID := grant["grant_id"].(string)
 revokedBefore := countGrantAudit(f.store.Read(), "authorization.revoke")
 f.request("demo-admin", "DELETE", grantPath+"/"+grantID, nil, 200)
 f.request("demo-admin", "DELETE", grantPath+"/"+grantID, nil, 200)
 if countGrantAudit(f.store.Read(), "authorization.revoke") != revokedBefore+1 { t.Fatal("repeated revoke duplicated audit") }
 if grantFlowVisible(t, f, id) { t.Fatal("revoked copy remained visible") }
 f.request("demo-user", "POST", sessionPath, map[string]any{}, 403)
 if !grantFlowVisible(t, f, "channel-replenishment") { t.Fatal("revoking copy removed unrelated application access") }
}
func TestApplicationCapabilityGrantDoesNotReplaceUserApplicationGrant(t *testing.T) {
 f := setup(t)
 id := publishGrantFlowCopy(t, f)
 capability := decode(t, f.request("demo-admin", "POST", grantPath, grantPayload("APPLICATION", id, "SERVICE", "replenishment-gap", "service.invoke"), 200))
 if grantFlowVisible(t, f, id) { t.Fatal("application capability grant implicitly authorized a user") }
 user := decode(t, f.request("demo-admin", "POST", grantPath, grantPayload("USER", "demo-user", "APPLICATION", id, "application.use"), 200))
 f.request("demo-admin", "DELETE", grantPath+"/"+capability["grant_id"].(string), nil, 200)
 if !grantFlowVisible(t, f, id) { t.Fatal("revoking service grant removed independent user use grant") }
 f.request("demo-admin", "DELETE", grantPath+"/"+user["grant_id"].(string), nil, 200)
 if grantFlowVisible(t, f, id) { t.Fatal("revoked user grant still exposed the copy") }
 // No business execution is submitted in this visibility test.
 for _, session := range f.store.Read().Sessions {
  raw, _ := json.Marshal(session)
  if strings.Contains(string(raw), id) { t.Fatal("visibility checks unexpectedly created a session") }
 }
}

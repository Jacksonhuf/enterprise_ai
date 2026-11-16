package api_test

import (
 "context"
 "encoding/json"
 "reflect"
 "testing"
 app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func TestTemplateCopyBindsPreviewVersionAndCreatesIndependentDraft(t *testing.T) {
 f := setup(t)
 before := f.store.Read()
 source := before.Apps["channel-replenishment"]
 body := map[string]any{"creation_mode":"TEMPLATE","name":"独立副本","description":"复制验证","template_id":source.ID,"expected_template_revision":source.Revision}
 var copied app.App
 if err := json.Unmarshal(f.request("demo-engineer","POST","/api/v1/cognitive/apps",body,200), &copied); err != nil { t.Fatal(err) }
 expected := source.Draft
 expected.Basic = app.Basic{Name:"独立副本",Description:"复制验证"}
 if copied.ID == source.ID || copied.AgentID == source.AgentID || copied.OwnerID != "demo-engineer" || copied.TenantID != source.TenantID || copied.Revision == "" || copied.CurrentReleaseID != nil || len(copied.Releases) != 0 || !reflect.DeepEqual(copied.Draft,expected) { t.Fatal("copy must be a fresh independent draft",copied) }
 after := f.store.Read()
 if len(after.Apps) != len(before.Apps)+1 || !reflect.DeepEqual(after.Grants,before.Grants) || !reflect.DeepEqual(after.Sessions,before.Sessions) || !reflect.DeepEqual(after.Apps[source.ID],source) { t.Fatal("copy inherited permissions, sessions, or changed source") }
 data,_ := json.Marshal(copied.Draft)
 var update map[string]any
 json.Unmarshal(data,&update)
 update["expected_draft_revision"] = copied.Revision
 update["model_prompt"] = map[string]any{"model_id":"validation-only","prompt":"独立编辑"}
 f.request("demo-engineer","PUT","/api/v1/cognitive/apps/"+copied.ID,update,200)
 if !reflect.DeepEqual(f.store.Read().Apps[source.ID],source) { t.Fatal("editing copy mutated source") }
}

func TestTemplateCopyRejectsStalePreviewAndKeepsLegacyRequestsCompatible(t *testing.T) {
 f := setup(t)
 source := f.store.Read().Apps["channel-replenishment"]
 data,_ := json.Marshal(source.Draft)
 var update map[string]any
 json.Unmarshal(data,&update)
 update["expected_draft_revision"] = source.Revision
 update["model_prompt"] = map[string]any{"model_id":"validation-only","prompt":"新模板提示词"}
 saved := decode(t,f.request("demo-engineer","PUT","/api/v1/cognitive/apps/"+source.ID,update,200))
 count := len(f.store.Read().Apps)
 body := map[string]any{"creation_mode":"TEMPLATE","name":"复制","template_id":source.ID,"expected_template_revision":source.Revision}
 f.request("demo-engineer","POST","/api/v1/cognitive/apps",body,409)
 if len(f.store.Read().Apps) != count { t.Fatal("stale preview created an application") }
 body["expected_template_revision"] = saved["draft_revision"]
 f.request("demo-engineer","POST","/api/v1/cognitive/apps",body,200)
 delete(body,"expected_template_revision")
 var legacy app.App
 json.Unmarshal(f.request("demo-engineer","POST","/api/v1/cognitive/apps",body,200),&legacy)
 if legacy.Draft.ModelPrompt.Prompt != "新模板提示词" { t.Fatal("legacy copy must use the current template draft") }
}

func TestTemplateCopyChecksTenantBeforeVersionAndValidatesRevisionInput(t *testing.T) {
 f := setup(t)
 source := f.store.Read().Apps["channel-replenishment"]
 if err := f.store.UpdateOperationalContext(context.Background(),func(st *app.State) error {
  foreign := source
  foreign.ID = "foreign-template"
  foreign.TenantID = "999"
  st.Apps[foreign.ID] = foreign
  return nil
 }); err != nil { t.Fatal(err) }
 count := len(f.store.Read().Apps)
 for _, id := range []string{"foreign-template","missing-template"} {
  f.request("demo-engineer","POST","/api/v1/cognitive/apps",map[string]any{"creation_mode":"TEMPLATE","name":"Denied","template_id":id,"expected_template_revision":"unknown"},404)
 }
 f.request("demo-user","POST","/api/v1/cognitive/apps",map[string]any{"creation_mode":"TEMPLATE","name":"Denied","template_id":source.ID,"expected_template_revision":source.Revision},403)
 f.request("demo-engineer","POST","/api/v1/cognitive/apps",map[string]any{"creation_mode":"BLANK","name":"Invalid","expected_template_revision":source.Revision},400)
 for _, revision := range []any{"",17} {
  f.request("demo-engineer","POST","/api/v1/cognitive/apps",map[string]any{"creation_mode":"TEMPLATE","name":"Invalid","template_id":source.ID,"expected_template_revision":revision},400)
 }
 if len(f.store.Read().Apps) != count { t.Fatal("denied creation wrote a draft") }
}

import {test} from "node:test";
import assert from "node:assert/strict";
import {buildPublishReview} from "./publishReview";
import {emptyConfig} from "./applicationDraft";
import type {Application} from "./types";
const app=():Application=>{
 const baseline={...emptyConfig(),basic:{name:"Original",description:"Live"}};
 return {app_id:"app",agent_id:"agent",tenant_id:"42",owner_id:"user",draft:JSON.parse(JSON.stringify(baseline)),draft_revision:"r2",current_release_id:"live1",
 releases:[{release_id:"live1",app_id:"app",config_snapshot:baseline,release_note:"",published_at:"",published_by:"user"}],updated_at:""};
};
test("Publish review compares to the current release and retains no mutable draft or release pointers",()=>{
 const a=app();a.draft.basic.name="New";const review=buildPublishReview(a,a.draft);
 assert.equal(review.appId,"app");assert.equal(review.revision,"r2");assert.equal(review.currentReleaseId,"live1");
 assert.deepEqual(review.rows,[{label:"应用名称",before:"Original",after:"New"}]);
 a.draft.basic.name="Later";a.releases[0].config_snapshot.basic.name="Mutated";a.draft_revision="r3";
 assert.equal(review.rows[0].before,"Original");assert.equal(review.rows[0].after,"New");assert.equal(review.revision,"r2");
});
test("First publication shows all nine configuration groups including empty values",()=>{
 const a=app();a.current_release_id=null;const review=buildPublishReview(a,a.draft);
 assert.equal(review.firstPublication,true);assert.equal(review.rows.length,9);
 assert.ok(review.rows.every(row=>row.before==="（尚未发布）"));
 assert.equal(review.rows.find(row=>row.label==="主模型")?.after,"validation-only");
 assert.equal(review.rows.find(row=>row.label==="欢迎语")?.after,"");
});
test("Missing, duplicate, cross-application and malformed live snapshots are never treated as empty baselines",()=>{
 for(const mode of ["missing","duplicate","cross","malformed"]){
  const a=app();
  if(mode==="missing")a.releases=[];
  if(mode==="duplicate")a.releases.push(a.releases[0]);
  if(mode==="cross")a.releases[0].app_id="other";
  if(mode==="malformed")a.releases[0].config_snapshot=null as any;
  assert.throws(()=>buildPublishReview(a,a.draft));
 }
});
test("Publish review preserves service versions and question order while ignoring reference set ordering",()=>{
 const a=app(),live=a.releases[0].config_snapshot;
 live.capability_refs.asset_refs=[{asset_id:"b"},{asset_id:"a"}];
 live.capability_refs.service_refs=[{service_id:"gap",version:"v1"}];
 live.conversation_defaults.suggested_questions=["One","Two"];
 a.draft=JSON.parse(JSON.stringify(live));a.draft.capability_refs.asset_refs.reverse();
 assert.equal(buildPublishReview(a,a.draft).rows.length,0);
 a.draft.capability_refs.service_refs[0].version="v2";a.draft.conversation_defaults.suggested_questions.reverse();
 const review=buildPublishReview(a,a.draft);
 assert.deepEqual(review.rows.map(row=>row.label),["认知服务版本","示例问题"]);
 assert.equal(review.rows[0].before,"gap · v1");assert.equal(review.rows[0].after,"gap · v2");
});

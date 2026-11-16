import {test} from "node:test";
import assert from "node:assert/strict";
import {reactive} from "vue";
import {ReleaseRestore} from "./releaseRestore";
import {emptyConfig} from "./applicationDraft";
import {APIError} from "./api";
import type {Application,Release} from "./types";
const app=():Application=>({app_id:"app",agent_id:"agent",tenant_id:"42",owner_id:"user",draft:emptyConfig(),draft_revision:"r1",current_release_id:"live2",releases:[],updated_at:""});
const release=():Release=>({app_id:"app",release_id:"live1",config_snapshot:{...emptyConfig(),basic:{name:"Historical",description:""}},release_note:"",published_at:"",published_by:"user"});
test("Restore captures a cloned historical config and current CAS revision without publishing",async()=>{
 let payload:any,id="";const a=app(),r=release();
 const state=reactive(new ReleaseRestore(async(key,value)=>{id=key;payload=value;return {...a,draft:value,draft_revision:"r2"}}));
 assert.equal(state.open(a,r),true);a.draft_revision="external";r.config_snapshot.basic.name="mutated";
 assert.equal(state.changes[0].after,"Historical");const result=await state.confirm();
 assert.equal(payload.expected_draft_revision,"r1");assert.equal(payload.basic.name,"Historical");assert.equal(id,"app");
 assert.equal(result?.current_release_id,"live2");assert.equal(state.visible,false);
});
test("Restore refuses cross-application and identical snapshots",()=>{
 const state=new ReleaseRestore(async()=>app()),r=release();r.app_id="other";
 assert.equal(state.open(app(),r),false);assert.match(state.error,/不匹配/);
 r.app_id="app";r.config_snapshot=emptyConfig();assert.equal(state.open(app(),r),false);assert.equal(state.visible,false);
});
test("Restore serializes submissions and ignores results after navigation",async()=>{
 let resolve!:(a:Application)=>void,calls=0;
 const state=new ReleaseRestore(async()=>{calls++;return new Promise(r=>resolve=r)});
 state.open(app(),release());const pending=state.confirm();state.close();
 assert.equal(state.visible,true);assert.equal(state.open(app(),release()),false);assert.equal(await state.confirm(),null);
 state.invalidate();resolve({...app(),draft:release().config_snapshot,draft_revision:"r2"});
 assert.equal(await pending,null);assert.equal(calls,1);assert.equal(state.visible,false);
});
test("Restore blocks retries after CAS conflicts and uncertain network outcomes",async()=>{
 for(const failure of [new APIError(409,"VERSION_INCOMPATIBLE","updated"),new Error("offline"),new APIError(503,"UNAVAILABLE","busy")]){
  let calls=0;const state=new ReleaseRestore(async()=>{calls++;throw failure});state.open(app(),release());
  assert.equal(await state.confirm(),null);assert.equal(state.blocked,true);assert.equal(await state.confirm(),null);
  state.close();assert.equal(state.open(app(),release()),false);assert.equal(calls,1);assert.match(state.error,/重新加载/);
 }
});
test("Restore rejects malformed or mismatched successful responses as uncertain",async()=>{
 for(const result of [{...app(),draft_revision:"r2"},{...app(),app_id:"other",draft_revision:"r2"},{...app(),draft:release().config_snapshot}]){
  const state=new ReleaseRestore(async()=>result);state.open(app(),release());assert.equal(await state.confirm(),null);
  assert.equal(state.blocked,true);assert.match(state.error,/结果未确认/);
 }
});

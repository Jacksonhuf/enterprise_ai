import {test} from "node:test";
import assert from "node:assert/strict";
import {reactive} from "vue";
import {APIError} from "./api";
import {GrantMutation,type GrantInput} from "./grantMutation";
import type {Grant} from "./types";
const input=():GrantInput=>({subject_type:"USER",subject_id:"demo-user",resource_type:"APPLICATION",resource_id:"new-app",action:"application.use",scope:"east"});
const rows=()=>[{label:"授权主体",value:"张敏"},{label:"目标资源",value:"业务助手"}];
const granted=(body=input()):Grant=>({...body,grant_id:"grant-new",active:true});
const transport=(overrides:any={})=>({create:async(body:GrantInput)=>granted(body),revoke:async()=>({success:true}),...overrides});
const deferred=<T>()=>{let resolve!:(value:T)=>void;const promise=new Promise<T>(r=>resolve=r);return {resolve,promise}};

test("Grant review captures its DTO and labels, strips extra fields and validates the accepted grant",async()=>{
 let sent:any;const draft:any={...input(),tenant_id:"injected"},labels=rows(),state=reactive(new GrantMutation(transport({create:async body=>{sent=body;return granted(body)}})));
 assert.equal(state.openCreate(draft,labels),true);draft.resource_id="other";labels[1].value="Changed";
 assert.equal(state.rows[1].value,"业务助手");assert.equal((await state.confirm() as Grant)?.resource_id,"new-app");
 assert.equal(sent.resource_id,"new-app");assert.equal(Object.hasOwn(sent,"tenant_id"),false);assert.equal(state.visible,false);
});
test("Grant requests are serialized and cannot be replaced or closed while saving",async()=>{
 const pending=deferred<Grant>();let calls=0;const state=new GrantMutation(transport({create:async()=>{calls++;return pending.promise}}));
 state.openCreate(input(),rows());const sending=state.confirm();state.close();assert.equal(state.visible,true);
 assert.equal(state.openRevoke(granted(),rows()),false);assert.equal(await state.confirm(),null);
 pending.resolve(granted());assert.ok(await sending);assert.equal(calls,1);
});
test("Revocation captures the selected grant and ignores responses after leaving the page",async()=>{
 const pending=deferred<{success:boolean}>();let id="";const grant=granted(),state=new GrantMutation(transport({revoke:async value=>{id=value;return pending.promise}}));
 state.openRevoke(grant,rows());grant.grant_id="other";const sending=state.confirm();state.invalidate();pending.resolve({success:true});
 assert.equal(await sending,null);assert.equal(id,"grant-new");assert.equal(state.visible,false);
 const active=new GrantMutation(transport());active.openRevoke(granted(),rows());assert.deepEqual(await active.confirm(),{success:true});
});
test("Network, server and malformed results require reloading instead of automatic retry",async()=>{
 for(const mode of ["network","server","mismatch","inactive","revoke"]){
  let calls=0;const state=new GrantMutation(transport({
   create:async()=>{calls++;if(mode==="network")throw Error("offline");if(mode==="server")throw new APIError(503,"UNAVAILABLE","busy");return mode==="inactive"?{...granted(),active:false}:{...granted(),resource_id:"other"}},
   revoke:async()=>{calls++;return {success:false}}
  }));
  if(mode==="revoke")state.openRevoke(granted(),rows());else state.openCreate(input(),rows());
  assert.equal(await state.confirm(),null);assert.equal(state.blocked,true);assert.match(state.error,/结果未确认/);
  assert.equal(await state.confirm(),null);state.close();assert.equal(state.openCreate(input(),rows()),false);assert.equal(calls,1);
 }
});
test("Invalid input and definite validation failures preserve a correctable form without granting",async()=>{
 let calls=0;const state=new GrantMutation(transport({create:async()=>{calls++;throw new APIError(400,"INVALID_REQUEST","unsupported")}}));
 assert.equal(state.openCreate({...input(),subject_id:" "},rows()),false);assert.equal(calls,0);
 assert.equal(state.openRevoke({...granted(),active:false},rows()),false);
 state.openCreate(input(),rows());assert.equal(await state.confirm(),null);assert.equal(state.blocked,false);assert.match(state.error,/unsupported/);
 state.close();assert.equal(state.openCreate({...input(),resource_id:"corrected"},rows()),true);
});
test("Permission and version failures block stale grant actions until an explicit list reload",async()=>{
 for(const status of [401,403,404,409]){
  const state=new GrantMutation(transport({revoke:async()=>{throw new APIError(status,"DENIED","changed","request_grant")}}));
  state.openRevoke(granted(),rows());assert.equal(await state.confirm(),null);assert.equal(state.blocked,true);
  assert.match(state.error,/request_grant/);state.close();assert.equal(state.openRevoke(granted(),rows()),false);
 }
});

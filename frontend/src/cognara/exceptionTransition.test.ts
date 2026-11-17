import {test} from "node:test";
import assert from "node:assert/strict";
import {reactive} from "vue";
import {APIError} from "./api";
import {ExceptionTransition,exceptionTransitions,type ExceptionSnapshot} from "./exceptionTransition";
const record=():ExceptionSnapshot=>({id:"exception-1",revision:"r1",status:"OPEN",name:"连接测试失败",request_id:"request-1"});
const result=(status:ExceptionSnapshot["status"]="ACKNOWLEDGED"):ExceptionSnapshot=>({...record(),revision:"r2",status});
const deferred=<T>()=>{let resolve!:(value:T)=>void;const promise=new Promise<T>(r=>resolve=r);return {resolve,promise}};

test("Exception state actions match the allowed lifecycle without exposing mutable shared actions",()=>{
 assert.deepEqual(exceptionTransitions("OPEN").map(a=>a.status),["ACKNOWLEDGED","RESOLVED"]);
 assert.deepEqual(exceptionTransitions("ACKNOWLEDGED").map(a=>a.status),["RESOLVED"]);
 assert.deepEqual(exceptionTransitions("RESOLVED").map(a=>a.status),["OPEN"]);
 assert.deepEqual(exceptionTransitions("UNKNOWN"),[]);
 const changed=exceptionTransitions("OPEN");changed[0].status="OPEN";assert.equal(exceptionTransitions("OPEN")[0].status,"ACKNOWLEDGED");
});
test("Exception confirmation captures record, version, request and target instead of mutable source data",async()=>{
 let sent:any,id="";const source=record(),state=reactive(new ExceptionTransition(async(key,body)=>{id=key;sent=body;return result()}));
 assert.equal(state.open(source,"ACKNOWLEDGED"),true);source.id="other";source.revision="new";source.request_id="other-request";
 state.record!.revision="view-mutation";state.targetStatus="RESOLVED";
 const updated=await state.confirm();assert.equal(updated?.status,"ACKNOWLEDGED");
 assert.equal(id,"exception-1");assert.deepEqual(sent,{expected_revision:"r1",status:"ACKNOWLEDGED"});assert.equal(state.visible,false);assert.equal(await state.confirm(),null);
});
test("Duplicate exception submissions and target changes are blocked; discarded pages ignore late responses",async()=>{
 const pending=deferred<ExceptionSnapshot>();let calls=0;const state=new ExceptionTransition(async()=>{calls++;return pending.promise});
 state.open(record(),"ACKNOWLEDGED");const sending=state.confirm();state.close();assert.equal(state.visible,true);
 assert.equal(state.open({...record(),id:"other"},"RESOLVED"),false);assert.equal(await state.confirm(),null);
 state.invalidate();pending.resolve(result());assert.equal(await sending,null);assert.equal(state.visible,false);assert.equal(calls,1);
});
test("Conflicts, uncertain errors and mismatched successful updates require a fresh list before retry",async()=>{
 for(const mode of ["conflict","network","server","id","revision","status","request"]){
  let calls=0;const state=new ExceptionTransition(async()=>{
   calls++;if(mode==="conflict")throw new APIError(409,"VERSION_INCOMPATIBLE","changed","trace_exception");
   if(mode==="network")throw Error("offline");if(mode==="server")throw new APIError(503,"UNAVAILABLE","busy");
   if(mode==="id")return {...result(),id:"other"};
   if(mode==="revision")return {...result(),revision:"r1"};
   if(mode==="status")return result("RESOLVED");
   return {...result(),request_id:"other-request"};
  });
  state.open(record(),"ACKNOWLEDGED");assert.equal(await state.confirm(),null);assert.equal(state.blocked,true);assert.match(state.error,/重新加载/);
  assert.equal(await state.confirm(),null);state.close();assert.equal(state.open(record(),"ACKNOWLEDGED"),false);assert.equal(calls,1);
 }
});
test("Invalid or canceled transitions cannot submit; reloading permits a reviewed new version",async()=>{
 let calls=0;const state=new ExceptionTransition(async()=>{calls++;return result("RESOLVED")});
 assert.equal(state.open({...record(),revision:""},"ACKNOWLEDGED"),false);
 assert.equal(state.open(record(),"OPEN"),false);assert.equal(state.open(record(),"INVALID"),false);
 state.open(record(),"RESOLVED");state.close();assert.equal(await state.confirm(),null);assert.equal(calls,0);
 state.blocked=true;assert.equal(state.open(record(),"RESOLVED"),false);
 state.blocked=false;assert.equal(state.open(record(),"RESOLVED"),true);assert.equal((await state.confirm())?.status,"RESOLVED");assert.equal(calls,1);
});

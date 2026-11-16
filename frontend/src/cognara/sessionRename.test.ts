import test from "node:test";
import assert from "node:assert/strict";
import { reactive } from "vue";
import { SessionRename, type SessionSummary } from "./sessionRename";
import { APIError } from "./api";
const target:SessionSummary={session_id:"one",app_id:"app",title:"Original",revision:"r1"};
test("Session rename validates Unicode title, snapshots versions and blocks duplicate submissions",async()=>{
 let resolve!:(v:SessionSummary)=>void;let calls=0;
 const draft=reactive(new SessionRename(async(id,payload)=>{calls++;assert.equal(id,"one");assert.equal(payload.expected_revision,"r1");assert.equal(payload.title,"新名称");return new Promise(r=>resolve=r)}));
 draft.open(target);draft.title=" ";assert.equal(await draft.save(),false);
 draft.title="字".repeat(81);assert.equal(await draft.save(),false);assert.equal(calls,0);
 draft.title=" 新名称 ";const pending=draft.save();assert.equal(await draft.save(),false);assert.equal(calls,1);
 resolve({...target,title:"新名称",revision:"r2"});assert.equal(await pending,true);assert.equal(draft.visible,false);
});
test("Session rename keeps conflicted input and requires reopening a refreshed record",async()=>{
 let calls=0;const draft=new SessionRename(async()=>{calls++;throw new APIError(409,"REVISION_CONFLICT","重新加载","trace_rename")});
 draft.open(target);draft.title="Unsaved";await draft.save();assert.equal(draft.title,"Unsaved");assert.equal(draft.conflict,true);assert.match(draft.error,/trace_rename/);
 await draft.save();assert.equal(calls,1);draft.close();draft.open({...target,revision:"r2"});assert.equal(draft.conflict,false);
});
test("A late rename response cannot close or mutate the dialog for a different session",async()=>{
 let resolve!:(v:SessionSummary)=>void;const draft=new SessionRename(()=>new Promise(r=>resolve=r));
 draft.open(target);draft.title="Changed";const pending=draft.save();draft.close();draft.open({...target,session_id:"two",title:"Other"});
 resolve({...target,revision:"r2"});assert.equal(await pending,false);assert.equal(draft.visible,true);assert.equal(draft.title,"Other");assert.equal(draft.saving,false);
});

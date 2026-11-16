import { test } from "node:test";
import assert from "node:assert/strict";
import { reactive } from "vue";
import { APIError } from "./api";
import { SettingsDraft,settingsValues,type SettingsSnapshot } from "./settingsDraft";
const deferred=()=>{let resolve!:(s:SettingsSnapshot)=>void;const promise=new Promise<SettingsSnapshot>(r=>{resolve=r});return{resolve,promise}};
test("Settings drafts block saves before successful load and isolate editable fields",async()=>{
 let writes=0;const state=reactive(new SettingsDraft("profile",async()=>{throw new Error("offline")},async()=>{writes++;return{values:{},revision:"r2"}}));
 assert.equal(await state.save(),false);await state.load();state.values.display_name="attempt";assert.equal(await state.save(),false);assert.equal(writes,0);
 assert.deepEqual(settingsValues("profile",{display_name:"Name",notice:"other partition",role:"admin"}),{display_name:"Name"});
 assert.deepEqual(settingsValues("general",{language:"zh-CN",timezone:42}),{language:"zh-CN",timezone:"",density:""});
});
test("Settings conflicts retain input, prevent blind retries and recover through explicit reload",async()=>{
 let writes=0;const state=reactive(new SettingsDraft("profile",async()=>({values:{display_name:"Server"},revision:"r1"}),async payload=>{writes++;assert.equal(payload.expected_revision,"r1");throw new APIError(409,"VERSION_INCOMPATIBLE","配置已变更","trace_settings")}));
 await state.load();state.values.display_name="My draft";assert.equal(await state.save(),false);
 assert.equal(state.values.display_name,"My draft");assert.equal(state.conflict,true);assert.equal(state.canSave,false);assert.match(state.error,/trace_settings/);
 await state.save();assert.equal(writes,1);await state.load();assert.equal(state.conflict,false);assert.equal(state.dirty,false);assert.equal(state.values.display_name,"Server");
});
test("Settings saves bind a version and do not overwrite edits made during a pending request",async()=>{
 const pending=deferred();let payload:any;
 const state=new SettingsDraft("profile",async()=>({values:{display_name:"Before"},revision:"r1"}),async value=>{payload=value;return pending.promise});
 await state.load();state.values.display_name="Saved";const save=state.save();assert.equal(await state.save(),false);state.values.display_name="Newer local edit";
 pending.resolve({values:{display_name:"Saved"},revision:"r2"});assert.equal(await save,true);
 assert.deepEqual(payload,{values:{display_name:"Saved"},expected_revision:"r1"});assert.equal(state.values.display_name,"Newer local edit");assert.equal(state.revision,"r2");assert.equal(state.canSave,true);
});
test("Settings stale loads are discarded and unversioned responses cannot enable save",async()=>{
 const old=deferred();let calls=0;
 const state=new SettingsDraft("profile",async()=>++calls===1?old.promise:{values:{display_name:"Latest"},revision:"r2"},async()=>({values:{},revision:"r3"}));
 const first=state.load();await state.load();old.resolve({values:{display_name:"Old"},revision:"r1"});await first;assert.equal(state.values.display_name,"Latest");assert.equal(state.revision,"r2");
 const missing=new SettingsDraft("profile",async()=>({values:{},revision:""}),async()=>({values:{},revision:"r2"}));await missing.load();assert.equal(missing.loaded,false);assert.equal(missing.canSave,false);assert.match(missing.error,/版本/);
});

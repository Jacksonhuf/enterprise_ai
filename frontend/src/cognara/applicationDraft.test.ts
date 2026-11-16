import {test} from "node:test";
import assert from "node:assert/strict";
import {reactive} from "vue";
import {APIError} from "./api";
import {ApplicationDraft,emptyConfig,type DraftTransport,type ReleaseCheck} from "./applicationDraft";
import type {Application,Release} from "./types";
const clone=<T>(v:T):T=>JSON.parse(JSON.stringify(v));
const app=():Application=>{
 const draft={...emptyConfig(),basic:{name:"App",description:""}};
 return {app_id:"app",agent_id:"agent",tenant_id:"42",owner_id:"user",draft,draft_revision:"r1",current_release_id:"live1",
 releases:[{release_id:"live1",app_id:"app",config_snapshot:clone(draft),release_note:"",published_at:"2026-11-16T00:00:00Z",published_by:"user"}],updated_at:"2026-11-16T00:00:00Z"};
};
const check=(revision="r1",expiry=Date.now()+60000):ReleaseCheck=>({status:"PASS",check_id:"check1",draft_revision:revision,valid_until:new Date(expiry).toISOString(),issues:[]});
const release=(config=app().draft):Release=>({release_id:"live2",app_id:"app",config_snapshot:clone(config),release_note:"",published_at:"2026-11-16T00:00:00Z",published_by:"user"});
const transport=(overrides:Partial<DraftTransport>={}):DraftTransport=>({read:async()=>app(),save:async payload=>{const {expected_draft_revision,...config}=payload;return {...app(),draft:clone(config),draft_revision:"r2"}},check:async payload=>check(payload.expected_draft_revision),publish:async()=>release(),...overrides});
const deferred=<T>()=>{let resolve!:(value:T)=>void;const promise=new Promise<T>(r=>{resolve=r});return{resolve,promise}};
test("Application editor load failures remain explicit and cannot save or publish",async()=>{
 const state=reactive(new ApplicationDraft(transport({read:async()=>{throw new Error("offline")}})));await state.load();assert.equal(state.application,null);assert.equal(state.canSave,false);assert.equal(state.canCheck,false);assert.equal(state.canPublishAt(Date.now()),false);assert.match(state.error,/offline/);assert.equal(state.loading,false);
 const malformed=new ApplicationDraft(transport({read:async()=>({...app(),draft:null} as unknown as Application)}));await malformed.load();assert.equal(malformed.application,null);assert.equal(malformed.canCheck,false);
 const empty=app();empty.releases=null as any;empty.current_release_id=null;empty.draft.capability_refs.asset_refs=null as any;const nullable=new ApplicationDraft(transport({read:async()=>empty}));await nullable.load();assert.deepEqual(nullable.config.capability_refs.asset_refs,[]);assert.equal(nullable.dirty,false);await nullable.checkPublish();assert.equal((await nullable.publish(""))?.release_id,"live2");
});
test("Application save clones reactive input, serializes requests and preserves edits made in flight",async()=>{
 const pending=deferred<Application>();let sent:any,calls=0;const state=reactive(new ApplicationDraft(transport({save:async value=>{sent=clone(value);calls++;return pending.promise}})));
 await state.load();state.config.basic.name="Sent";const saving=state.save();assert.equal(await state.save(),false);state.config.basic.name="New local edit";
 pending.resolve({...app(),draft:{...emptyConfig(),basic:{name:"Sent",description:""}},draft_revision:"r2"});assert.equal(await saving,true);assert.equal(sent.expected_draft_revision,"r1");assert.equal(sent.basic.name,"Sent");assert.equal(state.config.basic.name,"New local edit");assert.equal(state.dirty,true);assert.equal(state.application?.draft_revision,"r2");assert.equal(calls,1);
});
test("Application version conflicts retain draft and block overwrites until reload",async()=>{
 const state=new ApplicationDraft(transport({save:async()=>{throw new APIError(409,"VERSION_INCOMPATIBLE","updated","trace_app")}}));await state.load();state.config.basic.name="Mine";assert.equal(await state.save(),false);assert.equal(state.config.basic.name,"Mine");assert.equal(state.conflict,true);assert.equal(state.canSave,false);assert.match(state.error,/trace_app/);await state.load();assert.equal(state.conflict,false);assert.equal(state.dirty,false);
});
test("Release checks bind revision, local snapshot, release pointer and deadline",async()=>{
 const expiry=Date.now()+60000,state=new ApplicationDraft(transport({check:async()=>check("r1",expiry)}));await state.load();assert.equal(await state.checkPublish(),true);assert.equal(state.canPublishAt(expiry-1),true);assert.equal(state.canPublishAt(expiry),false);state.application!.draft_revision="other";assert.equal(state.canPublishAt(Date.now()),false);state.application!.draft_revision="r1";state.application!.current_release_id="other";assert.equal(state.canPublishAt(Date.now()),false);state.application!.current_release_id="live1";
 state.config.basic.name="Changed";assert.equal(state.canPublishAt(Date.now()),false);assert.equal(await state.publish(""),null);
});
test("Edits during check discard the result and missing-model errors remain correctable",async()=>{
 const pending=deferred<ReleaseCheck>(),state=new ApplicationDraft(transport({check:async()=>pending.promise}));await state.load();const checking=state.checkPublish();state.config.basic.name="New edit";pending.resolve(check());assert.equal(await checking,false);assert.equal(state.checkResult,null);
 const model=new ApplicationDraft(transport({check:async()=>{throw new APIError(409,"REFERENCE_NOT_READY","model unavailable")}}));await model.load();assert.equal(await model.checkPublish(),false);assert.equal(model.conflict,false);model.config.model_prompt.model_id="replacement";assert.equal(model.canSave,true);
});
test("Publish consumes checks and does not hide local edits or silently retry uncertain results",async()=>{
 const pending=deferred<Release>();let calls=0;const state=new ApplicationDraft(transport({publish:async()=>{calls++;return pending.promise}}));await state.load();await state.checkPublish();const sending=state.publish("note");state.config.basic.name="Not published";pending.resolve(release());assert.equal((await sending)?.release_id,"live2");assert.equal(state.dirty,true);assert.equal(state.checkResult,null);assert.equal(await state.publish("retry"),null);assert.equal(calls,1);
 const failed=new ApplicationDraft(transport({publish:async()=>{throw new Error("connection lost")}}));await failed.load();await failed.checkPublish();assert.equal(await failed.publish(""),null);assert.equal(failed.checkResult,null);assert.match(failed.error,/connection lost/);
});
test("Dirty drafts are saved before checking, but fresh edits stop automatic check continuation",async()=>{
 const pending=deferred<Application>();let checks=0;const state=new ApplicationDraft(transport({save:async()=>pending.promise,check:async()=>{checks++;return check("r2")}}));await state.load();state.config.basic.name="Snapshot";const checking=state.checkPublish();state.config.basic.name="Later";pending.resolve({...app(),draft:{...emptyConfig(),basic:{name:"Snapshot",description:""}},draft_revision:"r2"});assert.equal(await checking,false);assert.equal(checks,0);assert.equal(state.config.basic.name,"Later");
});
test("Late loads cannot replace a newer editor snapshot",async()=>{
 const old=deferred<Application>();let count=0;const state=new ApplicationDraft(transport({read:async()=>++count===1?old.promise:{...app(),draft_revision:"new"}}));const first=state.load();await state.load();old.resolve(app());await first;assert.equal(state.application?.draft_revision,"new");
});

test("Discarded editor instances ignore late publication results",async()=>{
 const pending=deferred<Release>(),state=new ApplicationDraft(transport({publish:async()=>pending.promise}));await state.load();await state.checkPublish();const publishing=state.publish("note");state.invalidate();pending.resolve(release());assert.equal(await publishing,null);assert.equal(state.application?.current_release_id,"live1");
});

test("Publication review is captured with the checked revision and cleared on save, reload and publish",async()=>{
 const state=new ApplicationDraft(transport());await state.load();state.config.basic.name="Reviewed";
 await state.checkPublish();assert.equal(state.review?.revision,"r2");assert.equal(state.review?.currentReleaseId,"live1");
 assert.deepEqual(state.review?.rows.find(row=>row.label==="应用名称"),{label:"应用名称",before:"App",after:"Reviewed"});
 state.application!.releases[0].config_snapshot.basic.name="External mutation";
 assert.equal(state.review?.rows[0].before,"App");
 state.config.basic.name="Later";assert.equal(state.canPublishAt(Date.now()),false);assert.equal(state.review?.rows[0].after,"Reviewed");
 await state.save();assert.equal(state.review,null);await state.checkPublish();await state.load();assert.equal(state.review,null);
 await state.checkPublish();assert.ok(state.review);await state.publish("");assert.equal(state.review,null);
});
test("Missing current release baselines fail before check requests; first publication is explicitly reviewable",async()=>{
 let checks=0;const missing=app();missing.releases=[];
 const state=new ApplicationDraft(transport({read:async()=>missing,check:async()=>{checks++;return check()}}));
 await state.load();assert.equal(await state.checkPublish(),false);assert.equal(checks,0);assert.equal(state.review,null);
 assert.match(state.error,/核对当前发布/);assert.equal(state.canPublishAt(Date.now()),false);
 const first=app();first.current_release_id=null;first.releases=[];
 const fresh=new ApplicationDraft(transport({read:async()=>first}));await fresh.load();await fresh.checkPublish();
 assert.equal(fresh.review?.firstPublication,true);assert.equal(fresh.review?.rows.length,9);assert.equal(fresh.canPublishAt(Date.now()),true);
});
test("Release pointer changes during checking discard the review and cannot enable publishing",async()=>{
 const pending=deferred<ReleaseCheck>(),state=new ApplicationDraft(transport({check:async()=>pending.promise}));
 await state.load();const checking=state.checkPublish();state.application!.current_release_id="live3";
 pending.resolve(check());assert.equal(await checking,false);assert.equal(state.review,null);assert.equal(state.checkResult,null);
 assert.equal(state.canPublishAt(Date.now()),false);
});
test("Uncertain publication and invalidated checks clear their review without automatic retry",async()=>{
 const state=new ApplicationDraft(transport({publish:async()=>{throw new Error("offline")}}));await state.load();await state.checkPublish();
 assert.ok(state.review);assert.equal(await state.publish(""),null);assert.equal(state.review,null);assert.equal(state.canPublishAt(Date.now()),false);
 const pending=deferred<ReleaseCheck>(),disposed=new ApplicationDraft(transport({check:async()=>pending.promise}));
 await disposed.load();const checking=disposed.checkPublish();disposed.invalidate();pending.resolve(check());
 assert.equal(await checking,false);assert.equal(disposed.review,null);assert.equal(disposed.checkResult,null);
});

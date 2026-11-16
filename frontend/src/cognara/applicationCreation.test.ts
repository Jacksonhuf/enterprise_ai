import {test} from "node:test";
import assert from "node:assert/strict";
import {reactive} from "vue";
import {ApplicationCreation,type CreateRequest} from "./applicationCreation";
import {emptyConfig} from "./applicationDraft";
import {APIError} from "./api";
import {loadApplicationOptions} from "./applicationOptions";
import type {Application} from "./types";
const clone=<T>(value:T):T=>JSON.parse(JSON.stringify(value));
const source=(id="source",revision="r1"):Application=>({app_id:id,agent_id:"agent",tenant_id:"42",owner_id:"owner",draft:{...emptyConfig(),basic:{name:"Source",description:"Template"},model_prompt:{model_id:"validation-only",prompt:"Safe prompt"}},draft_revision:revision,current_release_id:"live1",releases:[],updated_at:""});
const created=(body:CreateRequest):Application=>({...source("new","new-r1"),draft:{...clone(source().draft),basic:{name:body.name,description:body.description}},current_release_id:null,releases:[]});
const transport=(overrides:any={})=>({list:async()=>[source()],read:async(id:string)=>source(id),create:async(body:CreateRequest)=>created(body),...overrides});
const deferred=<T>()=>{let resolve!:(value:T)=>void;const promise=new Promise<T>(r=>resolve=r);return {resolve,promise}};
test("Template options use all independent management pages and discard partial lists on error",async()=>{
 const cursors:(string|null)[]=[];
 const state=new ApplicationCreation(transport({list:()=>loadApplicationOptions(async cursor=>{cursors.push(cursor);return cursor===null?{items:[source("a")],next_cursor:"next"}:{items:[source("b")],next_cursor:null}})}));
 state.open();state.mode="TEMPLATE";await state.loadTemplates();assert.deepEqual(cursors,[null,"next"]);assert.deepEqual(state.options.map(a=>a.app_id),["a","b"]);
 const failed=new ApplicationCreation(transport({list:()=>loadApplicationOptions(async cursor=>{if(cursor!==null)throw Error("offline");return {items:[source()],next_cursor:"next"}})}));
 failed.open();failed.mode="TEMPLATE";await failed.loadTemplates();assert.deepEqual(failed.options,[]);assert.match(failed.error,/offline/);
});
test("Template creation captures a cloned preview revision and sends only the selected source version",async()=>{
 let sent:any;const original=source(),state=reactive(new ApplicationCreation(transport({read:async()=>original,create:async(body:CreateRequest)=>{sent=clone(body);return created(body)}})));
 state.open();state.mode="TEMPLATE";state.name="Copy";state.description="Business";await state.selectTemplate("source");
 original.draft_revision="external";original.draft.model_prompt.prompt="Mutated";
 assert.equal(state.template?.draft.model_prompt.prompt,"Safe prompt");
 const result=await state.create();assert.equal(sent.expected_template_revision,"r1");assert.equal(sent.template_id,"source");
 assert.equal(result?.current_release_id,null);assert.equal(result?.draft.basic.name,"Copy");assert.equal(state.visible,false);
});
test("Late preview selections and discarded pages cannot restore an obsolete source",async()=>{
 const pending=deferred<Application>(),state=new ApplicationCreation(transport({read:async(id:string)=>id==="old"?pending.promise:source(id)}));
 state.open();state.mode="TEMPLATE";const old=state.selectTemplate("old");await state.selectTemplate("new");pending.resolve(source("old"));await old;
 assert.equal(state.template?.app_id,"new");
 const late=deferred<Application>(),closed=new ApplicationCreation(transport({read:async()=>late.promise}));
 closed.open();closed.mode="TEMPLATE";const loading=closed.selectTemplate("source");closed.invalidate();late.resolve(source());await loading;
 assert.equal(closed.template,null);assert.equal(closed.visible,false);
});
test("Template conflicts retain input and require an explicit refreshed preview before retry",async()=>{
 let calls=0,revision="r1";const state=new ApplicationCreation(transport({read:async()=>source("source",revision),create:async(body:CreateRequest)=>{calls++;if(calls===1)throw new APIError(409,"VERSION_INCOMPATIBLE","changed");assert.equal(body.expected_template_revision,"r2");return created(body)}}));
 state.open();state.mode="TEMPLATE";state.name="Mine";await state.selectTemplate("source");
 assert.equal(await state.create(),null);assert.equal(state.name,"Mine");assert.equal(state.template,null);assert.equal(state.canCreate,false);
 assert.equal(await state.create(),null);assert.equal(calls,1);
 revision="r2";await state.selectTemplate("source");assert.equal((await state.create())?.app_id,"new");assert.equal(calls,2);
});
test("Creation serializes requests, snapshots form data and ignores navigation results",async()=>{
 const pending=deferred<Application>();let sent!:CreateRequest,calls=0;
 const state=new ApplicationCreation(transport({create:async(body:CreateRequest)=>{sent=body;calls++;return pending.promise}}));
 state.open();state.name="Sent";const sending=state.create();state.name="Later";state.close();assert.equal(state.visible,true);
 assert.equal(await state.create(),null);state.invalidate();pending.resolve(created(sent));assert.equal(await sending,null);assert.equal(calls,1);assert.equal(sent.name,"Sent");
});
test("Uncertain or malformed creation results block duplicate retries and reopening",async()=>{
 for(const failure of ["network","server","malformed","mismatch"]){
  let calls=0;const state=new ApplicationCreation(transport({create:async(body:CreateRequest)=>{calls++;if(failure==="network")throw Error("offline");if(failure==="server")throw new APIError(503,"UNAVAILABLE","busy");return failure==="malformed"?{}:{...created(body),draft:{...emptyConfig(),basic:{name:"Wrong",description:""}}}}}));
  state.open();state.name="Mine";assert.equal(await state.create(),null);assert.equal(state.blocked,true);assert.match(state.error,/结果未确认/);
  assert.equal(await state.create(),null);state.close();state.open();assert.equal(state.visible,false);assert.equal(calls,1);
 }
});
test("Blank creation validates Unicode names and omits template fields while retaining correctable errors",async()=>{
 let sent!:CreateRequest;const state=new ApplicationCreation(transport({create:async(body:CreateRequest)=>{sent=body;throw new APIError(400,"INVALID_REQUEST","correct it")}}));
 state.open();state.name=" ";assert.equal(state.canCreate,false);state.name="𠮷".repeat(80);assert.equal(state.canCreate,true);
 state.name+="𠮷";assert.equal(state.canCreate,false);state.name="Valid";await state.create();
 assert.equal(sent.creation_mode,"BLANK");assert.equal(Object.hasOwn(sent,"template_id"),false);assert.equal(Object.hasOwn(sent,"expected_template_revision"),false);
 assert.equal(state.name,"Valid");assert.equal(state.blocked,false);assert.equal(state.canCreate,true);
});
test("Mismatched preview metadata and closed catalogue requests never enable copying",async()=>{
 const state=new ApplicationCreation(transport({read:async()=>source("other")}));
 state.open();state.mode="TEMPLATE";state.name="Copy";await state.selectTemplate("source");assert.equal(state.template,null);assert.equal(state.canCreate,false);assert.match(state.error,/有效版本/);
 const pending=deferred<Application[]>(),closed=new ApplicationCreation(transport({list:async()=>pending.promise}));
 closed.open();closed.mode="TEMPLATE";const loading=closed.loadTemplates();closed.close();pending.resolve([source()]);await loading;assert.deepEqual(closed.options,[]);
});

import { test } from "node:test";
import assert from "node:assert/strict";
import { reactive } from "vue";
import { APIError } from "./api";
import { MessageHistory, type MessagePage } from "./messageHistory";
import type { Message } from "./types";
const row=(id:string):Message=>({id,session_id:"session",role:"user",content:id,created_at:"2026-11-15T12:00:00Z",is_completed:true});
const page=(ids:string[],cursor:string|null):MessagePage=>({success:true,data:ids.map(row),has_more:cursor!==null,next_before_message_id:cursor});
const deferred=()=>{let resolve!:(p:MessagePage)=>void;const promise=new Promise<MessagePage>(r=>{resolve=r});return {resolve,promise};};
test("Message history prepends pages in server order, deduplicates overlap and stops at beginning",async()=>{
 let calls=0;const state=reactive(new MessageHistory(async cursor=>{calls++;return cursor===null?page(["3","4"],"3"):page(["1","2","3"],null)}));
 await state.refresh(); await state.older(); await state.older();
 assert.deepEqual(state.items.map(m=>m.id),["1","2","3","4"]); assert.equal(calls,2);
});
test("Message history refresh discards old in-flight pages and clears prior content immediately",async()=>{
 const old=deferred();let calls=0;const state=new MessageHistory(async()=>++calls===1?page(["2"],"2"):calls===2?old.promise:page(["current"],null));
 await state.refresh();const pending=state.older();const refreshed=state.refresh();
 assert.deepEqual(state.items,[]);await refreshed;
 old.resolve(page(["old"],null));await pending;
 assert.deepEqual(state.items.map(m=>m.id),["current"]);assert.equal(state.loading,false);
});
test("Message history retries transport failures but clears records on authorization loss",async()=>{
 let calls=0;const state=new MessageHistory(async()=>{calls++;if(calls===1)return page(["2"],"2");if(calls===2)throw new Error("network");throw new APIError(403,"AUTHORIZATION_DENIED","权限已撤销");});
 await state.refresh();await state.older();assert.equal(state.cursor,"2");assert.equal(state.items.length,1);assert.equal(state.error,"network");
 await state.older();assert.deepEqual(state.items,[]);assert.equal(state.cursor,null);assert.match(state.error,/撤销/);
});
test("Message history suppresses parallel loads and rejects a nonadvancing cursor",async()=>{
 const pending=deferred();let calls=0;const state=new MessageHistory(async()=>++calls===1?page(["2"],"2"):pending.promise);
 await state.refresh();const load=state.older();await state.older();assert.equal(calls,2);
 pending.resolve(page(["1"],"2"));await load;assert.match(state.error,/未前进/);assert.deepEqual(state.items.map(m=>m.id),["2"]);
});

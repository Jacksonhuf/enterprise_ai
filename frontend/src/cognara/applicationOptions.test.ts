import { test } from "node:test";
import assert from "node:assert/strict";
import type { Application } from "./types";
import { loadApplicationOptions } from "./applicationOptions";
const app=(id:string)=>({app_id:id} as Application);
test("Authorization options load beyond the first page and deduplicate overlap",async()=>{
 const requests:(string|null)[]=[];
 const apps=await loadApplicationOptions(async cursor=>{requests.push(cursor);return cursor===null?{items:Array.from({length:100},(_,i)=>app(String(i))),next_cursor:"100"}:{items:[app("99"),app("100"),app("101")],next_cursor:null}});
 assert.deepEqual(requests,[null,"100"]);assert.equal(apps.length,102);assert.equal(apps.at(-1)?.app_id,"101");
});
test("Authorization options fail as a whole on a later page and can be retried",async()=>{
 await assert.rejects(loadApplicationOptions(async cursor=>{if(cursor!==null)throw new Error("network");return {items:[app("a")],next_cursor:"1"}}),/network/);
 const apps=await loadApplicationOptions(async()=>({items:[app("b")]}));assert.deepEqual(apps.map(a=>a.app_id),["b"]);
});
test("Authorization options reject repeating and cycling cursors",async()=>{
 for(const sequence of [["1","1"],["1","2","1"]]){let index=0;await assert.rejects(loadApplicationOptions(async()=>({items:[],next_cursor:sequence[index++]})),/未前进/);assert.equal(index,sequence.length)}
});

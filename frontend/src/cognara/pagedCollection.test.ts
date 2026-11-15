import { test } from "node:test";
import assert from "node:assert/strict";
import { reactive } from "vue";
import { PagedCollection, type Page } from "./pagedCollection";
type Item={id:string};
const deferred=()=>{let resolve!:(page:Page<Item>)=>void; const promise=new Promise<Page<Item>>(r=>{resolve=r});return {resolve,promise};};
test("Paged collection ignores stale searches and stale completion while used through Vue reactive", async()=>{
  const old=deferred(), latest=deferred();
  const state=reactive(new PagedCollection<Item>(q=>q==="old"?old.promise:latest.promise,x=>x.id));
  const first=state.reset("old"), second=state.reset("new");
  old.resolve({items:[{id:"secret-old"}],next_cursor:"20"}); await first;
  assert.equal(state.loading,true); assert.deepEqual(state.items,[]);
  latest.resolve({items:[{id:"new"}],next_cursor:null}); await second;
  assert.deepEqual(state.items,[{id:"new"}]); assert.equal(state.loading,false);
});
test("Paged collection retries failed pages, suppresses duplicate calls and deduplicates overlapping rows", async()=>{
  let attempts=0; const pending=deferred();
  const state=new PagedCollection<Item>(async(_q,cursor)=>{
    if(cursor===null)return {items:[{id:"1"}],next_cursor:"20"};
    attempts++; if(attempts===1)throw new Error("暂不可用"); return pending.promise;
  },x=>x.id);
  await state.reset(); await state.next();
  assert.equal(state.error,"暂不可用"); assert.equal(state.nextCursor,"20");
  const retry=state.next(); await state.next(); assert.equal(attempts,2);
  pending.resolve({items:[{id:"1"},{id:"2"}],next_cursor:null}); await retry;
  assert.deepEqual(state.items.map(x=>x.id),["1","2"]); assert.equal(state.error,"");
  await state.next(); assert.equal(attempts,2);
});
test("Paged collection rejects nonadvancing cursors without losing existing rows",async()=>{
  const state=new PagedCollection<Item>(async(_q,cursor)=>({items:[{id:cursor??"first"}],next_cursor:"20"}),x=>x.id);
  await state.reset(); await state.next(); assert.match(state.error,/未前进/);
  assert.deepEqual(state.items,[{id:"first"}]);
});

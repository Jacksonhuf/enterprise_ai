import { test } from "node:test";
import assert from "node:assert/strict";
import { serviceKey,toggleServiceVersion,resolveServiceSelection } from "./capabilitySelection";
import { loadAllOptions } from "./applicationOptions";
test("Choosing a service version replaces only that service and preserves unavailable existing references",()=>{
 const refs=[{service_id:"a",version:"v1"},{service_id:"a",version:"v2"},{service_id:"b",version:"v1"},{service_id:"withdrawn",version:"old"}];
 const selected=[serviceKey(refs[0]),serviceKey(refs[2]),serviceKey(refs[3])];
 const switched=toggleServiceVersion(selected,serviceKey(refs[1]),true,refs);
 assert.deepEqual(resolveServiceSelection(switched,refs),[refs[2],refs[3],refs[1]]);
 assert.deepEqual(selected,[serviceKey(refs[0]),serviceKey(refs[2]),serviceKey(refs[3])]);
 const removed=toggleServiceVersion(switched,serviceKey(refs[2]),false,refs);
 assert.deepEqual(resolveServiceSelection(removed,refs),[refs[3],refs[1]]);
});
test("Service keys are unambiguous and conflicting or missing versions cannot be silently committed",()=>{
 assert.notEqual(serviceKey({service_id:"a:b",version:"c"}),serviceKey({service_id:"a",version:"b:c"}));
 const refs=[{service_id:"a",version:"v1"},{service_id:"a",version:"v2"}];
 assert.throws(()=>resolveServiceSelection(refs.map(serviceKey),refs),/只能引用一个版本/);
 assert.throws(()=>resolveServiceSelection(["missing"],refs),/已变化/);
 assert.throws(()=>toggleServiceVersion([],"missing",true,refs),/不在/);
 const resolved=resolveServiceSelection([serviceKey(refs[0])],refs);resolved[0].version="changed";assert.equal(refs[0].version,"v1");
});
test("Capability option pagination keeps different versions of the same service",async()=>{
 const first={service_id:"a",version:"v1"},second={service_id:"a",version:"v2"};
 const result=await loadAllOptions(async cursor=>cursor===null?{items:[first],next_cursor:"2"}:{items:[first,second],next_cursor:null},serviceKey);
 assert.deepEqual(result,[first,second]);
});

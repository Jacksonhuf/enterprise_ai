import { test } from "node:test";
import assert from "node:assert/strict";
import { governanceParams, pagedGovernance, type GovernanceFilters } from "./governanceFilters";
const filters=():GovernanceFilters=>({query:"request_1",status:"OPEN",outcome:"failed",unit:"TOKEN",range:[]});
test("Governance filters serialize real instants and preserve opaque cursor",()=>{
 const f=filters();f.range=["2026-11-16T08:00:00+08:00","2026-11-16T08:30:00+08:00"];
 const p=governanceParams("audit",f,"opaque+/cursor");
 assert.equal(p.get("from_time"),"2026-11-16T00:00:00.000Z");assert.equal(p.get("to_time"),"2026-11-16T00:30:00.000Z");assert.equal(p.get("cursor"),"opaque+/cursor");assert.equal(p.get("outcome"),"failed");assert.equal(p.has("status"),false);assert.equal(p.has("unit"),false);
});
test("Governance sections send only applicable filters and support cleared dates",()=>{
 const f=filters();f.range=null;
 assert.equal(governanceParams("usage",f,null).get("unit"),"TOKEN");
 const grants=governanceParams("grants",{...f,status:"REVOKED",range:["bad"]},null);assert.equal(grants.get("status"),"REVOKED");assert.equal(grants.has("from_time"),false);
 assert.equal(governanceParams("exceptions",f,null).get("status"),"OPEN");
 assert.equal(pagedGovernance("users"),false);assert.equal(governanceParams("users",f,null).has("limit"),false);
});
test("Governance time filters reject partial, invalid or reversed ranges",()=>{
 for(const range of [["bad"],["bad","2026-11-16"],["2026-11-17","2026-11-16"]])assert.throws(()=>governanceParams("audit",{...filters(),range},null),/时间范围/);
});

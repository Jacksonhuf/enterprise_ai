import {test} from "node:test";
import assert from "node:assert/strict";
import {auditRequestId,auditTraceLocation} from "./auditTrace";
test("Audit links target a fixed governance page and preserve only a valid request identifier",()=>{
 const id="request_A-123";
 assert.equal(auditRequestId(id),id);assert.deepEqual(auditTraceLocation(id),{path:"/governance/audit",query:{request_id:id}});
 assert.equal(auditRequestId("a".repeat(128)).length,128);
});
test("Ambiguous, malformed and oversized request route values never become links or filters",()=>{
 for(const value of [undefined,null,["request_one","request_two"],{},17,"","with space","../audit","https://example.com","request?next=other","a".repeat(129)]){
  assert.equal(auditRequestId(value),"");assert.equal(auditTraceLocation(value),null);
 }
});

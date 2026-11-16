import { test } from "node:test";
import assert from "node:assert/strict";
import { createPinia,setActivePinia } from "pinia";
import { useCognaraAuth } from "./stores/auth";
const principal={tenant_id:"42",actor_user_id:"demo-user",name:"张敏",role:"user" as const,scope:"east"};
Object.defineProperty(globalThis,"localStorage",{configurable:true,value:{getItem:()=>"demo-user",removeItem:()=>{}}});
test("Saved profile refresh updates the account label using the authenticated identity endpoint",async t=>{
 setActivePinia(createPinia());const auth=useCognaraAuth();auth.user={...principal};
 t.mock.method(globalThis,"fetch",async(path:any)=>{assert.equal(path,"/api/v1/cognitive/me");return new Response(JSON.stringify({...principal,name:"新名称"}),{status:200})});
 await auth.refreshProfile();assert.equal(auth.user?.name,"新名称");assert.equal(auth.canBuild,false);assert.equal(auth.isAdmin,false);
});
test("A profile refresh cannot repopulate the account after logout",async t=>{
 setActivePinia(createPinia());const auth=useCognaraAuth();auth.user={...principal};
 let resolve!:(response:Response)=>void;
 t.mock.method(globalThis,"fetch",()=>new Promise<Response>(r=>{resolve=r}));
 const request=auth.refreshProfile();auth.logout();resolve(new Response(JSON.stringify({...principal,name:"Stale"}),{status:200}));await request;assert.equal(auth.user,null);
});

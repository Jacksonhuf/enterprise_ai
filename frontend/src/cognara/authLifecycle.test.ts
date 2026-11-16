import test from 'node:test';
import assert from 'node:assert/strict';
import { createPinia, setActivePinia } from 'pinia';
import { useCognaraAuth } from './stores/auth';
const user={tenant_id:'42',actor_user_id:'demo-user',name:'User',role:'user',scope:'east'};
function setup(){setActivePinia(createPinia());let identity='demo-user';Object.defineProperty(globalThis,'localStorage',{configurable:true,value:{getItem:()=>identity,setItem:(_k:string,v:string)=>{identity=v},removeItem:()=>{identity=''}}});return useCognaraAuth();}
const response=(data:unknown,status=200)=>new Response(JSON.stringify(data),{status});
function deferred(){let resolve!:(r:Response)=>void;const promise=new Promise<Response>(r=>resolve=r);return {promise,resolve};}
test('Logout invalidates a pending identity request and resets model state',async t=>{
 const auth=setup(),pending=deferred();t.mock.method(globalThis,'fetch',()=>pending.promise);
 auth.modelsConfigured=true;const load=auth.load();auth.logout();pending.resolve(response(user));await assert.rejects(load,/账户已变化/);assert.equal(auth.user,null);assert.equal(auth.modelsConfigured,false);
});
test('Logout invalidates a pending runtime response before publishing identity',async t=>{
 const auth=setup(),runtime=deferred();t.mock.method(globalThis,'fetch',async(path:any)=>String(path).endsWith('/me')?response(user):runtime.promise);
 const load=auth.load();await new Promise(r=>setTimeout(r,0));assert.equal(auth.user,null);auth.logout();runtime.resolve(response({models_configured:true}));await assert.rejects(load);assert.equal(auth.user,null);assert.equal(auth.modelsConfigured,false);
});
test('Failed previous login cannot clear a successfully initialized newer login',async t=>{
 const auth=setup(),old=deferred();let calls=0;
 t.mock.method(globalThis,'fetch',async(path:any)=>{if(++calls===1)return old.promise;return response(String(path).endsWith('/me')?{...user,actor_user_id:'demo-admin',role:'admin'}:{models_configured:true})});
 const first=auth.login('demo-user');const failed=assert.rejects(first);await auth.login('demo-admin');old.resolve(response({error:{message:'old failed'}},401));await failed;assert.equal(auth.user?.actor_user_id,'demo-admin');assert.equal(auth.isAdmin,true);assert.equal(localStorage.getItem('x'),'demo-admin');
});
test('Runtime failure leaves no partially authenticated identity or previous model status',async t=>{
 const auth=setup();auth.modelsConfigured=true;t.mock.method(globalThis,'fetch',async(path:any)=>String(path).endsWith('/me')?response(user):response({error:{message:'Unavailable'}},503));
 await assert.rejects(auth.login('demo-user'));assert.equal(auth.user,null);assert.equal(auth.modelsConfigured,false);assert.equal(localStorage.getItem('x'),'');
});
test('Same-account logout and login also invalidates an earlier profile refresh',async t=>{
 const auth=setup();auth.user=user as any;const old=deferred();let calls=0;
 t.mock.method(globalThis,'fetch',async(path:any)=>{if(++calls===1)return old.promise;return response(String(path).endsWith('/me')?{...user,name:'New'}:{models_configured:false})});
 const refresh=auth.refreshProfile();auth.logout();await auth.login('demo-user');old.resolve(response({...user,name:'Old'}));await refresh;assert.equal(auth.user?.name,'New');
});

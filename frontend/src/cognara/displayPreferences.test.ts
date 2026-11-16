import test from 'node:test';
import assert from 'node:assert/strict';
import { createPinia, setActivePinia } from 'pinia';
import { useCognaraAuth } from './stores/auth';
const user={tenant_id:'42',actor_user_id:'demo-user',name:'User',role:'user' as const,scope:'east'};
const response=(data:unknown,status=200)=>new Response(JSON.stringify(data),{status});
const snapshot=(density:string)=>({revision:'r1',values:{density,timezone:'America/New_York'}});
function setup(){setActivePinia(createPinia());let identity='demo-user';Object.defineProperty(globalThis,'localStorage',{configurable:true,value:{getItem:()=>identity,setItem:(_k:string,v:string)=>{identity=v},removeItem:()=>{identity=''}}});return useCognaraAuth();}
function deferred(){let resolve!:(r:Response)=>void;const promise=new Promise<Response>(r=>resolve=r);return {promise,resolve};}
test('Login restores server density and clearing it restores the standard layout',async t=>{
 const auth=setup();let density='compact';
 t.mock.method(globalThis,'fetch',async(path:any)=>response(String(path).endsWith('/me')?user:String(path).endsWith('/runtime-status')?{models_configured:false}:snapshot(density)));
 await auth.load();assert.equal(auth.density,'compact');assert.equal(auth.timezone,'America/New_York');density='';await auth.refreshPreferences();assert.equal(auth.density,'comfortable');assert.equal(auth.preferencesError,'');
});
test('Unavailable preferences do not block login, and retry can recover',async t=>{
 const auth=setup();let failed=true;
 t.mock.method(globalThis,'fetch',async(path:any)=>String(path).endsWith('/me')?response(user):String(path).endsWith('/runtime-status')?response({models_configured:false}):failed?response({error:{message:'暂时不可用'}},503):response(snapshot('compact')));
 await auth.login('demo-user');assert.equal(auth.user?.actor_user_id,'demo-user');assert.equal(auth.density,'comfortable');assert.ok(auth.preferencesError);failed=false;await auth.refreshPreferences();assert.equal(auth.density,'compact');assert.equal(auth.preferencesError,'');
});
test('Same-account relogin invalidates an older preference response',async t=>{
 const auth=setup();auth.user=user;auth.density='compact';const old=deferred();let first=true;
 t.mock.method(globalThis,'fetch',async(path:any)=>{if(first){first=false;return old.promise}return response(String(path).endsWith('/me')?user:String(path).endsWith('/runtime-status')?{models_configured:false}:snapshot('comfortable'))});
 const pending=auth.refreshPreferences();auth.logout();assert.equal(auth.density,'comfortable');assert.equal(auth.timezone,'');await auth.login('demo-user');old.resolve(response(snapshot('compact')));await pending;assert.equal(auth.density,'comfortable');assert.equal(auth.preferencesError,'');
});
test('An older refresh cannot replace the newest preference or error state',async t=>{
 const auth=setup();auth.user=user;const old=deferred();let first=true;
 t.mock.method(globalThis,'fetch',async()=>{if(first){first=false;return old.promise}return response(snapshot('compact'))});
 const pending=auth.refreshPreferences();await auth.refreshPreferences();old.resolve(response({error:{message:'old failure'}},503));await pending;assert.equal(auth.density,'compact');assert.equal(auth.preferencesError,'');
});
test('Malformed preference responses preserve the applied density and allow retry',async t=>{
 const auth=setup();auth.user=user;auth.density='compact';auth.timezone='America/New_York';t.mock.method(globalThis,'fetch',async()=>response({values:[]}));await auth.refreshPreferences();assert.equal(auth.density,'compact');assert.equal(auth.timezone,'America/New_York');assert.ok(auth.preferencesError);
});
test('Logout while login preferences are pending cancels the original initialization',async t=>{
 const auth=setup(),pending=deferred();let started!:()=>void;const ready=new Promise<void>(r=>started=r);
 t.mock.method(globalThis,'fetch',async(path:any)=>{if(String(path).endsWith('/me'))return response(user);if(String(path).endsWith('/runtime-status'))return response({models_configured:true});started();return pending.promise});
 const login=auth.login('demo-user');const rejected=assert.rejects(login,/账户已变化/);await ready;auth.logout();pending.resolve(response(snapshot('compact')));await rejected;assert.equal(auth.user,null);assert.equal(auth.density,'comfortable');assert.equal(auth.preferencesError,'');
});

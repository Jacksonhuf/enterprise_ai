import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {runInNewContext} from 'node:vm';
import {patchRadioGroupLifecycle,tdesignLifecyclePatch} from './scripts-tdesign-lifecycle.mjs';
const upstream=readFileSync(new URL('./node_modules/tdesign-vue-next/es/radio/group.mjs',import.meta.url),'utf8');
function component(source) {
 const start=source.indexOf('    var radioGroupDisposed = false;')>=0?source.indexOf('    var radioGroupDisposed = false;'):source.indexOf('    var calcDefaultBarStyle =');
 const end=source.indexOf('    var observerReturn;',start);
 const callbacks=[],state={value:null},style={value:{}},document={createElement:()=>({setAttribute(){},appendChild(){},querySelector:()=>null}),body:{appendChild(){},removeChild(){document.removed++}} ,removed:0};
 const api=runInNewContext(source.slice(start,end)+'; ({measure:calcBarStyle,fallback:calcDefaultBarStyle})',{
  radioGroupRef:state,barStyle:style,checkedClassName:{value:'.checked'},props2:{variant:'default-filled'},
  onUnmounted:callback=>callbacks.push(callback),document,_objectSpread:Object.assign,
 });
 return {...api,state,style,document,unmount:()=>{callbacks.forEach(f=>f());state.value=null}};
}
test('Installed upstream reproduces the deferred RadioGroup null-ref failure; patch protects unmount and detached nodes',()=>{
 assert.throws(()=>component(upstream).measure(),/querySelector/);
 const group=component(patchRadioGroupLifecycle(upstream));
 assert.doesNotThrow(()=>group.measure());
 group.state.value={isConnected:false,querySelector(){throw Error('detached DOM measured')}};
 assert.doesNotThrow(()=>group.measure());assert.doesNotThrow(()=>group.fallback());
 group.unmount();assert.doesNotThrow(()=>group.measure());assert.doesNotThrow(()=>group.fallback());
});
test('Mounted RadioGroup retains selected geometry and animation',()=>{
 const group=component(patchRadioGroupLifecycle(upstream));
 group.state.value={isConnected:true,querySelector:()=>({offsetWidth:90,offsetHeight:32,offsetLeft:8,offsetTop:4})};
 group.measure();assert.equal(group.style.value.width,'90px');assert.equal(group.style.value.left,'8px');assert.equal(group.style.value['transition-property'],'all');
 group.measure(true);assert.equal(group.style.value['transition-property'],'none');
 group.unmount();group.state.value={isConnected:true,querySelector(){throw Error('disposed component measured')}};
 assert.doesNotThrow(()=>group.measure());
});
test('Hidden RadioGroup fallback removes its temporary DOM when no selected label remains',()=>{
 const group=component(patchRadioGroupLifecycle(upstream));
 group.state.value={isConnected:true,cloneNode:()=>({}),querySelector:()=>({offsetWidth:0})};
 assert.doesNotThrow(()=>group.measure());assert.equal(group.document.removed,1);
});
test('Patch stays scoped to pinned ESM RadioGroup and rejects unsupported source changes',()=>{
 const plugin=tdesignLifecyclePatch();assert.deepEqual(plugin.config().optimizeDeps.exclude,['tdesign-vue-next']);assert.equal(plugin.transform(upstream,'/node_modules/tdesign-vue-next/es/tabs/index.mjs'),null);
 assert.ok(plugin.transform(upstream,'/node_modules/tdesign-vue-next/es/radio/group.mjs').code.includes('radioGroupDisposed'));
 assert.throws(()=>patchRadioGroupLifecycle(upstream.replace('v1.19.2','v2.0.0')),/new TDesign version/);
 assert.throws(()=>patchRadioGroupLifecycle(upstream.replace('var calcDefaultBarStyle =','var changed =')),/source changed/);
});

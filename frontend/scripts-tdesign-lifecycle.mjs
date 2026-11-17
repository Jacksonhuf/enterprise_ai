// Compatibility patch for TDesign 1.19.2 RadioGroup (MIT).
// Preserve the upstream component and visual behavior. Deferred measurement must
// not dereference a ref cleared by Vue during unmount. Fail closed on upgrades.
export function patchRadioGroupLifecycle(source) {
  if (!source.includes('* tdesign v1.19.2')) throw Error('Review RadioGroup lifecycle patch for the new TDesign version');
  const replaceOnce = (before, after) => {
    if (source.split(before).length !== 2) throw Error('TDesign RadioGroup patch source changed: ' + before);
    source = source.replace(before, after);
  };
  replaceOnce('    var calcDefaultBarStyle = function calcDefaultBarStyle() {',
    '    var radioGroupDisposed = false;\n    onUnmounted(function () { radioGroupDisposed = true; });\n    var calcDefaultBarStyle = function calcDefaultBarStyle() {\n      if (radioGroupDisposed || !radioGroupRef.value || !radioGroupRef.value.isConnected) return;');
  replaceOnce('      var checkedRadio = radioGroupRef.value.querySelector(checkedClassName.value);',
    '      if (radioGroupDisposed || !radioGroupRef.value || !radioGroupRef.value.isConnected) return;\n      var checkedRadio = radioGroupRef.value.querySelector(checkedClassName.value);');
  replaceOnce('      var defaultCheckedRadio = div.querySelector(checkedClassName.value);',
    '      var defaultCheckedRadio = div.querySelector(checkedClassName.value);\n      if (!defaultCheckedRadio) { document.body.removeChild(div); return; }');
  return source;
}
export function tdesignLifecyclePatch() {
  return {
    name: 'cognara-tdesign-radio-lifecycle', enforce: 'pre',
    // Keep dev dependency prebundling from bypassing this source transform.
    config() { return {optimizeDeps: {exclude: ['tdesign-vue-next']}}; },
    transform(source, id) {
      if (!id.split('?')[0].replaceAll('\\', '/').endsWith('/tdesign-vue-next/es/radio/group.mjs')) return null;
      return {code: patchRadioGroupLifecycle(source), map: null};
    },
  };
}

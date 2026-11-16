import { readFile, stat } from 'node:fs/promises';
import assert from 'node:assert/strict';
const manifest=JSON.parse(await readFile('dist/.vite/manifest.json','utf8'));const seen=new Set();
async function initial(key){if(seen.has(key))return 0;seen.add(key);const item=manifest[key];let bytes=item.file.endsWith('.js')?(await stat('dist/'+item.file)).size:0;for(const dep of item.imports||[])bytes+=await initial(dep);return bytes;}
const startup=await initial('index.html');let total=0,largest=0;
for(const file of new Set(Object.values(manifest).map(x=>x.file).filter(x=>x.endsWith('.js')))){const size=(await stat('dist/'+file)).size;total+=size;largest=Math.max(largest,size);}
assert.ok(startup<500_000,`Startup JS exceeds 500 KB: ${startup}`);assert.ok(largest<500_000,`Single JS chunk exceeds 500 KB: ${largest}`);assert.ok(total<1_200_000,`Total JS exceeds 1.2 MB: ${total}`);
assert.ok(!Object.keys(manifest).some(x=>x.startsWith('src/api/auth/')),'Upstream authentication leaked into F/G');
console.log(JSON.stringify({startup_js_bytes:startup,largest_chunk_bytes:largest,total_js_bytes:total}));

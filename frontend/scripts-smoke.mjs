import { chromium } from 'playwright-core';
import { preview } from 'vite';
import { spawn } from 'node:child_process';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import assert from 'node:assert/strict';
const temp=await mkdtemp(join(tmpdir(),'cognara-browser-'));
const backend=spawn(process.env.COGNARA_SMOKE_BINARY||'/tmp/cognara-dev',['--demo','--port','18080','--mock-port','18081','--data',join(temp,'state.json')],{stdio:'ignore'});
let server,browser;
try {
 for(let i=0;;i++){try{if((await fetch('http://127.0.0.1:18080/ready')).ok)break;}catch{} if(i===100)throw Error('Backend did not become ready');await new Promise(r=>setTimeout(r,100));}
 server=await preview({preview:{port:14173,strictPort:true,proxy:{'/api':{target:'http://127.0.0.1:18080',changeOrigin:true}}}});
 browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE_PATH||'/usr/bin/chromium',args:['--no-sandbox']});
 const page=await browser.newPage();const failures=[];
 page.on('pageerror',e=>failures.push(e.message));page.on('console',m=>{if(m.type()==='error'||/Failed to resolve component|injection.*not found/.test(m.text()))failures.push(m.text());});
 await page.goto('http://127.0.0.1:14173/login');await page.getByRole('button',{name:'进入验证工作台'}).click();await page.waitForURL('**/apps');
 await page.getByRole('button',{name:'进入应用'}).first().click();await page.waitForURL('**/chat/**');
 const input=page.locator('textarea');await input.fill('SO1002需求增加到350');await input.press('Shift+Enter');assert.equal(await page.locator('.message').count(),0);
 await input.fill('SO1002需求增加到350');await input.press('Enter');await page.locator('.message.assistant').waitFor();assert.match(await page.locator('.message.assistant').innerText(),/276.6/);
 await page.goto('http://127.0.0.1:14173/apps');await page.evaluate(()=>localStorage.setItem('cognara_demo_identity','demo-admin'));
 const routes=['/apps','/build/apps','/build/apps/channel-replenishment','/build/apps/channel-replenishment/edit','/assets','/services','/governance/access','/governance/audit','/governance/operations','/settings/profile','/settings/general','/settings/models','/forbidden'];
 for(const route of routes){await page.goto('http://127.0.0.1:14173'+route);await page.locator('h1, .wk-empty-state__title').first().waitFor();await page.waitForLoadState('networkidle');assert.ok((await page.locator('h1, .wk-empty-state__title').first().innerText()).trim());}
 assert.deepEqual(failures,[]);console.log('PASS Chromium: login, chat Enter/Shift+Enter, business result, 13 routes, no component/runtime errors');
} finally {await browser?.close();if(server)await new Promise(r=>server.httpServer.close(r));backend.kill('SIGTERM');await new Promise(r=>backend.exitCode!==null?r():backend.once('exit',r));await rm(temp,{recursive:true,force:true});}

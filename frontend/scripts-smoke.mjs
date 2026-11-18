import { chromium } from 'playwright-core';
import { preview } from 'vite';
import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import assert from 'node:assert/strict';
const temp=await mkdtemp(join(tmpdir(),'cognara-browser-'));
const backend=spawn(process.env.COGNARA_SMOKE_BINARY||'/tmp/cognara-dev',['--demo','--port','18080','--mock-port','18081','--data',join(temp,'state.json')],{stdio:'ignore'});
let server,browser;
const r18CaptureDir=process.env.COGNARA_R18_CAPTURE;
const r18Manifest=[];
async function assertR18Spot(page,spot){
 const [w,h]=spot.viewport||[1280,720];
 await page.setViewportSize({width:w,height:h});
 let captureTarget;
 if(spot.publicLogin){
  await page.evaluate(()=>localStorage.removeItem('cognara_demo_identity'));
  await page.goto('http://127.0.0.1:14173/login');await page.waitForLoadState('networkidle');
  await page.getByRole('heading',{name:'登录 Cognara'}).waitFor();
  const loginText=await page.locator('.login-page').innerText();assert.doesNotMatch(loginText,/schema\s*[0-9]|R20\b|架构编号/i);
  captureTarget=page.locator('.login-page');
 }else{
  if(spot.identity)await page.evaluate(id=>localStorage.setItem('cognara_demo_identity',id),spot.identity);
  await page.goto('http://127.0.0.1:14173'+spot.path);await page.waitForLoadState('networkidle');
  if(spot.publishReview){
  await page.locator('h1, .wk-empty-state__title').first().waitFor();
  assert.ok(await page.locator('.app-shell').isVisible());
  await page.getByRole('button',{name:'检查并发布',exact:true}).click();
  await page.locator('.publish-review').waitFor();
  captureTarget=page.locator('.publish-review');
  }else if(spot.sessionSidebar){
  await page.locator('.application-card').first().getByRole('button',{name:'进入应用',exact:true}).click();
  await page.waitForURL('**/chat/**');await page.locator('textarea').waitFor();
  await page.getByRole('button',{name:'搜索 / 刷新历史',exact:true}).click();await page.waitForLoadState('networkidle');
  captureTarget=page.locator('.app-shell');
  }else if(spot.chatEntry){
  await page.locator('.application-card').first().getByRole('button',{name:'进入应用',exact:true}).click();
  await page.waitForURL('**/chat/**');await page.locator('textarea').waitFor();
  captureTarget=page.locator('.app-shell');
  }else if(spot.forbidden){
  await page.waitForURL('**/forbidden');await page.getByText('当前无法访问此功能',{exact:true}).waitFor();
  captureTarget=page.locator('.center-state');
  }else if(!spot.chatEntry && !spot.publishReview && !spot.sessionSidebar){
  await page.locator('h1, .wk-empty-state__title').first().waitFor();
  assert.ok(await page.locator('.app-shell').isVisible());await page.getByRole('button',{name:'账户菜单'}).waitFor();
  const shellText=await page.locator('.app-shell').innerText();assert.doesNotMatch(shellText,/schema\s*[0-9]|R20\b|架构编号/i);
  captureTarget=page.locator('.app-shell');
  }
 }
 if(r18CaptureDir){
  await mkdir(r18CaptureDir,{recursive:true});
  const file=join(r18CaptureDir,spot.slug+'.png');
  await captureTarget.screenshot({path:file});
  r18Manifest.push({slug:spot.slug,path:spot.path,viewport:[w,h],identity:spot.identity||'demo-admin',forbidden:!!spot.forbidden,publicLogin:!!spot.publicLogin,chatEntry:!!spot.chatEntry,publishReview:!!spot.publishReview,sessionSidebar:!!spot.sessionSidebar,file:spot.slug+'.png'});
 }
}
try {
 for(let i=0;;i++){try{if((await fetch('http://127.0.0.1:18080/ready')).ok)break;}catch{} if(i===100)throw Error('Backend did not become ready');await new Promise(r=>setTimeout(r,100));}
 server=await preview({preview:{port:14173,strictPort:true,proxy:{'/api':{target:'http://127.0.0.1:18080',changeOrigin:true}}}});
 browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE_PATH||'/usr/bin/chromium',args:['--no-sandbox']});
 const page=await browser.newPage({timezoneId:'UTC'});const failures=[],diagnostics=[];
 page.on('pageerror',e=>{
  failures.push(e.message);console.error('Browser pageerror at '+page.url()+'\\n'+e.stack);
  const location=e.stack?.match(/(http:\/\/127\.0\.0\.1:14173\/assets\/[^\s)]+\.js):(\d+):(\d+)/);
  if(location)diagnostics.push((async()=>{
   const source=await(await fetch(location[1])).text(),line=source.split(String.fromCharCode(10))[Number(location[2])-1]||'',column=Number(location[3])-1;
   console.error('Browser source context: '+line.slice(Math.max(0,column-350),column+350));
  })().catch(error=>console.error('Browser source lookup failed: '+error.message)));
 });page.on('console',m=>{if(m.type()==='error'||/Failed to resolve component|injection.*not found/.test(m.text()))failures.push(m.text());});
 await page.goto('http://127.0.0.1:14173/login');await page.getByRole('button',{name:'进入验证工作台'}).click();await page.waitForURL('**/apps');
 await page.getByRole('button',{name:'进入应用'}).first().click();await page.waitForURL('**/chat/**');
 const input=page.locator('textarea');await input.fill('SO1002需求增加到350');await input.press('Shift+Enter');assert.equal(await page.locator('.message').count(),0);
 await input.fill('SO1002需求增加到350');await input.press('Enter');await page.locator('.message.assistant').waitFor();assert.match(await page.locator('.message.assistant').innerText(),/276.6/);
 await page.goto('http://127.0.0.1:14173/apps');await page.evaluate(()=>localStorage.setItem('cognara_demo_identity','demo-admin'));
 const routes=['/apps','/build/apps','/build/apps/channel-replenishment','/build/apps/channel-replenishment/edit','/assets','/services','/governance/access','/governance/audit','/governance/operations','/settings/profile','/settings/general','/settings/models','/forbidden'];
 for(const route of routes){await page.goto('http://127.0.0.1:14173'+route);await page.locator('h1, .wk-empty-state__title').first().waitFor();await page.waitForLoadState('networkidle');assert.ok((await page.locator('h1, .wk-empty-state__title').first().innerText()).trim());}
 // Exercise RadioGroup observer callbacks while a resized open drawer is unmounted by SPA back.
 await page.goto('http://127.0.0.1:14173/apps');
 for(let cycle=0;cycle<8;cycle++){
  await page.locator('.sidebar a.nav-link[href="/build/apps"]').click();await page.waitForURL('**/build/apps');
  await page.getByRole('button',{name:'新建应用',exact:true}).first().click();
  await page.getByText('从现有应用复制',{exact:true}).click();
  await page.setViewportSize({width:1280+cycle*8,height:720});
  await page.goBack();await page.waitForURL('**/apps');
 }
 await page.setViewportSize({width:1280,height:720});
 // Inspect the published service contract without executing it or inventing output fields.
 await page.goto('http://127.0.0.1:14173/services');
 await page.locator('.resource-card').filter({hasText:'补货缺口分析'}).getByRole('button',{name:'详情',exact:true}).click();
 await page.getByText('gap-contract-v1',{exact:true}).waitFor();
 const contract=page.locator('.service-contract');
 const quantityRow=contract.getByRole('row').filter({has:page.getByText('target_quantity',{exact:true})});
 assert.match(await quantityRow.innerText(),/integer/);assert.match(await quantityRow.innerText(),/最大值：100000/);
 assert.match(await quantityRow.innerText(),/最小值：1/);
 await page.getByText('输出契约',{exact:true}).click();
 await contract.getByText('目录未提供具名字段明细，请查看原始定义。',{exact:true}).waitFor();
 assert.equal(await contract.getByRole('row').count(),2);
 await contract.locator('summary').click();
 assert.deepEqual(JSON.parse(await contract.locator('pre').innerText()),{type:'object'});
 await page.getByText('输入契约',{exact:true}).click();await quantityRow.waitFor();
 await page.goto('http://127.0.0.1:14173/assets');
 await page.locator('.resource-card').first().getByRole('button',{name:'详情',exact:true}).click();
 assert.equal(await page.locator('.service-contract').count(),0);
 // Change only the draft, then restore through the actual history drawer and dialog.
 const before=await page.evaluate(async()=>{
  const path='/api/v1/cognitive/apps/channel-replenishment',headers={'Content-Type':'application/json',Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')};
  const response=await fetch(path,{headers});if(!response.ok)throw Error('read draft failed');const app=await response.json();
  const changed=JSON.parse(JSON.stringify(app.draft));changed.basic.description='Browser restoration regression';
  const saved=await fetch(path,{method:'PUT',headers,body:JSON.stringify({...changed,expected_draft_revision:app.draft_revision})});
  if(!saved.ok)throw Error('prepare draft failed');return app;
 });
 await page.goto('http://127.0.0.1:14173/build/apps/channel-replenishment?tab=releases');
 await page.getByRole('button',{name:'查看配置与差异',exact:true}).first().click();
 await page.getByRole('button',{name:'恢复为草稿',exact:true}).click();
 await page.getByRole('button',{name:'确认恢复',exact:true}).click();
 await page.waitForURL('**/build/apps/channel-replenishment/edit');
 await page.reload();await page.waitForLoadState('networkidle');
 const after=await page.evaluate(async()=>{const r=await fetch('/api/v1/cognitive/apps/channel-replenishment',{headers:{Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')}});if(!r.ok)throw Error('verify restore failed');return r.json()});
 assert.deepEqual(after.draft,before.releases.find(r=>r.release_id===before.current_release_id).config_snapshot);
 assert.equal(after.current_release_id,before.current_release_id);assert.deepEqual(after.releases,before.releases);
 assert.notEqual(after.draft_revision,before.draft_revision);
 // Review the unchanged baseline, then publish a visible, version-bound change.
 await page.getByRole('button',{name:'检查并发布',exact:true}).click();
 const publicationReview=page.locator('.publish-review');
 await publicationReview.getByText('配置与当前发布一致，确认仍会产生新发布版本。',{exact:true}).waitFor();
 await page.getByRole('button',{name:'取消',exact:true}).click();
 await page.locator('.t-form__item').filter({hasText:'业务描述'}).locator('textarea').fill('发布审阅浏览器回归');
 await page.getByRole('button',{name:'检查并发布',exact:true}).click();
 const descriptionReview=publicationReview.getByRole('row').filter({has:page.getByText('业务描述',{exact:true})});
 await descriptionReview.waitFor();
 assert.equal(await descriptionReview.locator('td').nth(1).innerText(),after.draft.basic.description || '（空）');
 assert.equal(await descriptionReview.locator('td').nth(2).innerText(),'发布审阅浏览器回归');
 const checkedDraft=await page.evaluate(async()=>{const r=await fetch('/api/v1/cognitive/apps/channel-replenishment',{headers:{Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')}});if(!r.ok)throw Error('read checked draft failed');return r.json()});
 assert.equal(checkedDraft.current_release_id,after.current_release_id);assert.deepEqual(checkedDraft.releases,after.releases);
 await publicationReview.getByText(checkedDraft.draft_revision,{exact:true}).waitFor();
 await page.getByRole('button',{name:'确认发布',exact:true}).click();
 await page.waitForURL('**/build/apps/channel-replenishment');
 const publishedDraft=await page.evaluate(async()=>{const r=await fetch('/api/v1/cognitive/apps/channel-replenishment',{headers:{Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')}});if(!r.ok)throw Error('verify published draft failed');return r.json()});
 assert.notEqual(publishedDraft.current_release_id,after.current_release_id);
 assert.equal(publishedDraft.releases.length,after.releases.length+1);
 assert.deepEqual(publishedDraft.releases.slice(0,after.releases.length),after.releases);
 assert.equal(publishedDraft.releases.find(r=>r.release_id===publishedDraft.current_release_id).config_snapshot.basic.description,'发布审阅浏览器回归');
 // Template sources must remain selectable when the application table search is empty.
 await page.goto('http://127.0.0.1:14173/build/apps');
 await page.locator('.resource-search-toolbar input').fill('不存在的模板筛选条件');
 await page.getByText('未找到匹配应用',{exact:true}).waitFor();
 await page.getByRole('button',{name:'新建应用',exact:true}).first().click();
 const creationForm=page.locator('.creation-form');
 await creationForm.locator('.t-form__item').filter({hasText:'应用名称'}).locator('input').fill('模板复制浏览器回归');
 await creationForm.locator('.t-form__item').filter({hasText:'业务描述'}).locator('textarea').fill('独立草稿验证');
 const templateList=page.waitForResponse(r=>r.url().includes('/api/v1/cognitive/apps?')&&r.url().includes('purpose=MANAGE')&&r.url().includes('limit=100'));
 await page.getByText('从现有应用复制',{exact:true}).click();assert.equal((await templateList).status(),200);
 await creationForm.locator('.t-select').click();await page.getByText(publishedDraft.draft.basic.name,{exact:true}).click();
 await creationForm.locator('.creation-preview').getByText(publishedDraft.draft_revision,{exact:true}).waitFor();
 const creationRequest=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/api/v1/cognitive/apps'));
 await page.getByRole('button',{name:'创建并配置',exact:true}).click();
 const creationPayload=(await creationRequest).postDataJSON();
 assert.equal(creationPayload.template_id,'channel-replenishment');assert.equal(creationPayload.expected_template_revision,publishedDraft.draft_revision);
 await page.waitForURL('**/build/apps/*/edit');await page.waitForLoadState('networkidle');
 const copiedId=decodeURIComponent(new URL(page.url()).pathname.split('/')[3]);
 const copiedApp=await page.evaluate(async id=>{const r=await fetch('/api/v1/cognitive/apps/'+encodeURIComponent(id),{headers:{Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')}});if(!r.ok)throw Error('read copy failed');return r.json()},copiedId);
 assert.notEqual(copiedApp.app_id,publishedDraft.app_id);assert.notEqual(copiedApp.agent_id,publishedDraft.agent_id);
 assert.equal(copiedApp.current_release_id,null);assert.deepEqual(copiedApp.releases,[]);
 assert.deepEqual(copiedApp.draft,{...publishedDraft.draft,basic:{name:'模板复制浏览器回归',description:'独立草稿验证'}});
 const unchangedSource=await page.evaluate(async()=>{const r=await fetch('/api/v1/cognitive/apps/channel-replenishment',{headers:{Authorization:'Bearer '+localStorage.getItem('cognara_demo_identity')}});if(!r.ok)throw Error('read source failed');return r.json()});
 assert.deepEqual(unchangedSource,publishedDraft);
 // Publish the copy, then grant user access through the reviewed governance form.
 await page.getByRole('button',{name:'检查并发布',exact:true}).click();
 await page.locator('.publish-review').getByText('首次发布 · 尚无线上的发布版本',{exact:true}).waitFor();
 await page.getByRole('button',{name:'确认发布',exact:true}).click();
 await page.waitForURL('**/build/apps/'+copiedId);
 const beforeUserAccess=await page.evaluate(async()=>{const r=await fetch('/api/v1/cognitive/apps?purpose=USE&limit=100',{headers:{Authorization:'Bearer demo-user'}});if(!r.ok)throw Error('read user catalogue failed');return r.json()});
 assert.equal(beforeUserAccess.items.some(a=>a.app_id===copiedId),false);
 const grantOptions=page.waitForResponse(r=>r.url().includes('/api/v1/cognitive/apps?')&&r.url().includes('purpose=MANAGE')&&r.url().includes('limit=100'));
 await page.goto('http://127.0.0.1:14173/governance/access?tab=grants');assert.equal((await grantOptions).status(),200);
 await page.getByRole('button',{name:'新增授权',exact:true}).click();
 const grantForm=page.locator('.grant-form');
 await grantForm.locator('.t-form__item').first().locator('.t-select').click();
 await page.locator('.t-select-option').filter({hasText:/^用户$/}).click();
 await grantForm.locator('.t-form__item').nth(3).locator('.t-select').click();
 await page.locator('.t-select-option').filter({hasText:copiedApp.draft.basic.name}).click();
 await page.getByRole('button',{name:'审阅授权',exact:true}).click();
 const grantReview=page.locator('.grant-review');await grantReview.waitFor();
 assert.match(await grantReview.innerText(),/张敏/);assert.ok((await grantReview.innerText()).includes(copiedApp.draft.basic.name));
 const grantRequest=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/api/v1/cognitive/governance/grants'));
 const grantResponse=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/api/v1/cognitive/governance/grants'));
 await page.getByRole('button',{name:'确认授权',exact:true}).click();
 const grantBody=(await grantRequest).postDataJSON();
 assert.equal(grantBody.subject_type,'USER');assert.equal(grantBody.subject_id,'demo-user');
 assert.equal(grantBody.resource_id,copiedId);assert.equal(grantBody.action,'application.use');
 const grantHTTP=await grantResponse;assert.equal(grantHTTP.status(),200);const createdAccessGrant=await grantHTTP.json();
 await page.getByRole('row').filter({hasText:copiedApp.draft.basic.name}).first().waitFor();
 // A real missing-model failure creates the exception and correlated request audit.
 // Use Node fetch for this expected 503, so browser console failures stay unexpected.
 const failedModel=await fetch('http://127.0.0.1:18080/api/v1/cognitive/models/not-configured/test',{method:'POST',headers:{Authorization:'Bearer demo-admin','Content-Type':'application/json'},body:'{}'});
 assert.equal(failedModel.status,503);const failedEnvelope=await failedModel.json(),traceRequest=failedEnvelope.request_id;
 assert.ok(traceRequest);assert.equal(failedModel.headers.get('x-request-id'),traceRequest);
 const exceptionHTTP=await fetch('http://127.0.0.1:18080/api/v1/cognitive/governance/exceptions?query='+encodeURIComponent(traceRequest),{headers:{Authorization:'Bearer demo-admin'}});
 assert.equal(exceptionHTTP.status,200);const originException=(await exceptionHTTP.json()).items[0];assert.equal(originException.request_id,traceRequest);
 await page.goto('http://127.0.0.1:14173/governance/operations?tab=exceptions');
 const exceptionRow=page.getByRole('row').filter({hasText:traceRequest});await exceptionRow.waitFor();
 await exceptionRow.getByRole('button',{name:'详情',exact:true}).click();
 const correlatedAudit=page.waitForResponse(r=>r.url().includes('/api/v1/cognitive/governance/audit?')&&new URL(r.url()).searchParams.get('query')===traceRequest);
 await page.getByRole('button',{name:'查看关联审计',exact:true}).click();
 await page.waitForURL(url=>url.pathname==='/governance/audit'&&url.searchParams.get('request_id')===traceRequest);
 const auditHTTP=await correlatedAudit;assert.equal(auditHTTP.status(),200);const correlated=(await auditHTTP.json()).items;
 assert.ok(correlated.length);assert.ok(correlated.every(event=>event.request_id===traceRequest));
 assert.equal(await page.locator('.list-toolbar input').first().inputValue(),traceRequest);
 // Exercise SPA back/forward repeatedly so pending component cleanup cannot hide behind full reloads.
 for(let round=0;round<20;round++){
  await page.goBack();await exceptionRow.waitFor();
  if(round%2===0){
   await exceptionRow.getByRole('button',{name:'详情',exact:true}).click();
   await page.getByRole('button',{name:'查看关联审计',exact:true}).click();
  }else await exceptionRow.getByRole('button',{name:'关联审计',exact:true}).click();
  await page.waitForURL(url=>url.pathname==='/governance/audit'&&url.searchParams.get('request_id')===traceRequest);
 }
 await page.goBack();await exceptionRow.waitFor();
 let exceptionRevision=originException.revision;
 for(const [action,status,label] of [['确认','ACKNOWLEDGED','已确认'],['标记解决','RESOLVED','已解决'],['重新打开','OPEN','待处理']]){
  await exceptionRow.getByRole('button',{name:action,exact:true}).click();
  const transitionRequest=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/exceptions/'+originException.id+'/status'));
  const transitionResponse=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/exceptions/'+originException.id+'/status'));
  await page.getByRole('button',{name:'确认更新',exact:true}).click();
  const mutation=(await transitionRequest).postDataJSON();assert.equal(mutation.expected_revision,exceptionRevision);assert.equal(mutation.status,status);
  const response=await transitionResponse;assert.equal(response.status(),200);const updated=await response.json();
  assert.equal(updated.status,status);assert.equal(updated.request_id,traceRequest);assert.notEqual(updated.revision,exceptionRevision);exceptionRevision=updated.revision;
  await exceptionRow.getByText(label,{exact:true}).waitFor();
 }
 // Public configuration must show persisted text, without identity-label substitution or HTML.
 for(const notice of ['demo-user','企业公告 <img src=x onerror=alert(1)>']){
  await page.goto('http://127.0.0.1:14173/governance/operations?tab=settings');
  await page.getByRole('link',{name:'编辑公共配置',exact:true}).click();
  await page.waitForURL('**/settings/public');await page.waitForLoadState('networkidle');
  await page.locator('textarea').fill(notice);
  const noticeSaved=page.waitForResponse(r=>r.request().method()==='PUT'&&r.url().endsWith('/settings/public'));
  await page.getByRole('button',{name:'保存',exact:true}).click();assert.equal((await noticeSaved).status(),200);
  await page.goto('http://127.0.0.1:14173/governance/operations?tab=settings');
  const noticeRow=page.getByRole('row').filter({hasText:'企业公告'});
  await noticeRow.getByText(notice,{exact:true}).waitFor();assert.equal(await noticeRow.locator('img').count(),0);
  await page.reload();await noticeRow.getByText(notice,{exact:true}).waitFor();
 }
 // Exercise the saved preference through the actual form and a fresh page load.
 await page.goto('http://127.0.0.1:14173/settings/general');
 await page.waitForLoadState('networkidle');
 const densityField=page.locator('.t-form__item').filter({hasText:'界面密度'});
 await densityField.locator('.t-select').click();await page.getByText('紧凑',{exact:true}).click();
 await page.locator('.t-form__item').filter({hasText:'时区'}).locator('input').fill('America/New_York');
 await page.getByRole('button',{name:'保存',exact:true}).click();
 await page.waitForFunction(()=>document.querySelector('.app-shell')?.getAttribute('data-density')==='compact');
 await page.reload();await page.waitForFunction(()=>document.querySelector('.app-shell')?.getAttribute('data-density')==='compact');
 await page.goto('http://127.0.0.1:14173/build/apps');
 const displayedTime=page.locator('time').first();await displayedTime.waitFor();
 const instant=await displayedTime.getAttribute('datetime');
 assert.equal(await displayedTime.innerText(),new Intl.DateTimeFormat('zh-CN',{timeZone:'America/New_York',year:'numeric',month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',second:'2-digit',hourCycle:'h23',timeZoneName:'shortOffset'}).format(new Date(instant)));
 await page.goto('http://127.0.0.1:14173/apps');await page.locator('.application-card').first().waitFor();
 assert.equal(await page.locator('.application-card').first().evaluate(el=>getComputedStyle(el).paddingTop),'18px');
 await page.getByRole('button',{name:'账户菜单'}).click();await page.getByRole('button',{name:'退出当前账户'}).click();await page.waitForURL('**/login');
 await page.getByRole('button',{name:'进入验证工作台'}).click();await page.waitForURL('**/apps');assert.equal(await page.getByRole('link',{name:'身份与权限',exact:true}).count(),0);
 assert.equal(await page.locator('.app-shell').getAttribute('data-density'),'comfortable');
 await page.locator('.application-card').first().waitFor();assert.equal(await page.locator('.application-card').first().evaluate(el=>getComputedStyle(el).paddingTop),'24px');
 // User can enter after authorization; revocation removes only the copied application's access.
 const copiedCard=page.locator('.application-card').filter({hasText:copiedApp.draft.basic.name});
 await copiedCard.waitFor();await copiedCard.getByRole('button',{name:'进入应用',exact:true}).click();await page.waitForURL('**/chat/**');
 await page.goto('http://127.0.0.1:14173/apps');await page.evaluate(()=>localStorage.setItem('cognara_demo_identity','demo-admin'));
 await page.goto('http://127.0.0.1:14173/governance/access?tab=grants');
 const grantedRow=page.getByRole('row').filter({hasText:copiedApp.draft.basic.name});await grantedRow.waitFor();
 await grantedRow.getByRole('button',{name:'撤销',exact:true}).click();
 const revokeResponse=page.waitForResponse(r=>r.request().method()==='DELETE'&&r.url().endsWith('/governance/grants/'+createdAccessGrant.grant_id));
 await page.getByRole('button',{name:'确认撤销',exact:true}).click();const revoked=await revokeResponse;assert.equal(revoked.status(),200);assert.equal((await revoked.json()).success,true);
 await page.goto('http://127.0.0.1:14173/apps');await page.evaluate(()=>localStorage.setItem('cognara_demo_identity','demo-user'));
 await page.goto('http://127.0.0.1:14173/apps');await page.waitForLoadState('networkidle');
 assert.equal(await page.locator('.application-card').filter({hasText:copiedApp.draft.basic.name}).count(),0);
 assert.equal(await page.locator('.application-card').filter({hasText:publishedDraft.draft.basic.name}).count(),1);
 // R18 spot-page regression (v1.04): expanded surfaces + optional screenshot baseline.
 const r18Spots=[
  {slug:'login-public',path:'/login',publicLogin:true},
  {slug:'build-apps-engineer',path:'/build/apps',identity:'demo-engineer'},
  {slug:'assets-engineer',path:'/assets',identity:'demo-engineer'},
  {slug:'settings-tenant',path:'/settings/tenant'},
  {slug:'governance-ops-settings',path:'/governance/operations?tab=settings'},
  {slug:'settings-models',path:'/settings/models'},
  {slug:'governance-access',path:'/governance/access?tab=grants'},
  {slug:'governance-audit',path:'/governance/audit'},
  {slug:'forbidden-user',path:'/governance/access',identity:'demo-user',forbidden:true},
  {slug:'apps-narrow',path:'/apps',viewport:[390,844],identity:'demo-user'},
  {slug:'chat-user',path:'/apps',identity:'demo-user',chatEntry:true},
  {slug:'editor-channel',path:'/build/apps/channel-replenishment/edit',identity:'demo-engineer'},
  {slug:'publish-review',path:'/build/apps/channel-replenishment/edit',identity:'demo-engineer',publishReview:true},
  {slug:'app-detail-build',path:'/build/apps/channel-replenishment',identity:'demo-engineer'},
  {slug:'session-sidebar',path:'/apps',identity:'demo-user',sessionSidebar:true},
  {slug:'build-releases',path:'/build/apps/channel-replenishment?tab=releases',identity:'demo-engineer'},
  {slug:'governance-usage',path:'/governance/operations?tab=usage'},
  {slug:'governance-ops-exceptions',path:'/governance/operations?tab=exceptions'},
  {slug:'governance-ops-audit',path:'/governance/operations?tab=audit'},
  {slug:'governance-grants-narrow',path:'/governance/access?tab=grants',viewport:[390,844]},
  {slug:'governance-ops-exceptions-narrow',path:'/governance/operations?tab=exceptions',viewport:[390,844]},
 ];
 for(const spot of r18Spots)await assertR18Spot(page,spot);
 if(r18CaptureDir)await writeFile(join(r18CaptureDir,'manifest.json'),JSON.stringify({generated_at:new Date().toISOString(),spots:r18Manifest},null,2)+'\n');
 await Promise.all(diagnostics);assert.deepEqual(failures,[]);console.log('PASS Chromium: login, chat Enter/Shift+Enter, business result, 13 routes, R18 spot pages ('+r18Spots.map(s=>s.slug).join(', ')+'), 8 RadioGroup resize/drawer/SPA unmount cycles, literal public notice/reload, service input/output contract and raw definition, historical release restore/reload/live snapshot protection, checked publication diff/unchanged baseline/real publish, independent template selector/preview/versioned copy, reviewed user grant/entry/revocation isolation, real failure/audit trace/20 repeated SPA/detail transitions/exception lifecycle, saved density/timezone/reload/account isolation, no component/runtime errors');
} finally {await browser?.close();if(server)await new Promise(r=>server.httpServer.close(r));backend.kill('SIGTERM');await new Promise(r=>backend.exitCode!==null?r():backend.once('exit',r));await rm(temp,{recursive:true,force:true});}

const fs=require('fs'),path=require('path'),run=require('./browser-harness.cjs');
run(async({page,base,report,setRole,setSignedIn})=>{
 const checks=[];const check=(name,pass,details)=>checks.push({name,pass:Boolean(pass),details});
 const visit=async(route,role)=>{setRole(role);setSignedIn(true);await page.goto(base+'/?lifecycle='+Date.now()+'#'+route);await page.locator('.care-app').waitFor();};
 // Exercise real Axios/XHR cancellation against held mock responses.
 let customerMode='normal';const customerRequests=[];
 await page.route('**/api/v1/customer/select',route=>{
  customerRequests.push(route.request());
  if(customerMode==='hold')return;
  if(customerMode==='error')return route.fulfill({status:503,json:{}});
  return route.fulfill({json:{customers:[{customerId:1,customerName:'F02当前监护人',customerPhone:'19912345678',customerAge:70}],total:1}});
 });
 await visit('/customer','ADMIN');
 await page.getByText('F02当前监护人',{exact:true}).first().waitFor();
 const searchCustomer=async(name)=>{
  await page.getByLabel('监护人姓名',{exact:true}).fill(name);
  const requested=page.waitForRequest(r=>r.url().endsWith('/api/v1/customer/select')&&r.postDataJSON().customerName===name);
  await page.getByRole('button',{name:'查询监护人姓名',exact:true}).click();
  return requested;
 };
 customerMode='hold';const superseded=await searchCustomer('F02旧查询');
 const canceled=page.waitForEvent('requestfailed',{predicate:r=>r===superseded});
 customerMode='normal';await searchCustomer('F02新查询');await canceled;
 await page.locator('.roles[aria-busy="false"]').waitFor();
 check('New guardian query aborts a hung XHR and displays its own result',await page.getByText('F02当前监护人',{exact:true}).count()>0);
 check('Read cancellation preserves the CSRF header',customerRequests.every(r=>r.headers()['x-csrf-token']==='mock-csrf-token'));
 customerMode='hold';const timedOut=await searchCustomer('F02超时');
 const timeoutCanceled=page.waitForEvent('requestfailed',{predicate:r=>r===timedOut});
 await page.getByRole('alert').filter({hasText:'监护人信息暂时无法加载'}).waitFor({timeout:15000});await timeoutCanceled;
 check('Guardian timeout aborts XHR and ends loading',await page.locator('.roles').getAttribute('aria-busy')==='false');
 customerMode='normal';await searchCustomer('F02恢复');
 await page.getByText('F02当前监护人',{exact:true}).first().waitFor();
 check('A fresh query recovers after timeout',await page.getByRole('alert').filter({hasText:'监护人信息暂时无法加载'}).count()===0);
 customerMode='error';await searchCustomer('F02当前错误');
 await page.getByRole('alert').filter({hasText:'监护人信息暂时无法加载'}).waitFor();
 check('A current HTTP failure ends loading and remains visible',await page.locator('.roles').getAttribute('aria-busy')==='false');
 customerMode='hold';const leaving=await searchCustomer('F02离页');
 const unmounted=page.waitForEvent('requestfailed',{predicate:r=>r===leaving});
 await page.evaluate(()=>{location.hash='/main'});await unmounted;
 await page.getByRole('heading',{name:'运营总览',exact:true}).waitFor();
 const countAfterLeaving=customerRequests.length;await page.waitForTimeout(100);
 check('Navigation aborts the guardian read without starting a replacement',customerRequests.length===countAfterLeaving);
 await page.unroute('**/api/v1/customer/select');

 await page.route('**/api/v1/care/incidents',route=>route.fulfill({json:{items:[{id:1,oldPersonName:'F02照护任务',state:'received'}],total:1,duty:'supervisor',actorId:1}}));
 const historyRequests=[];
 await page.route('**/api/v1/care/events',route=>{historyRequests.push(route.request())});
 await visit('/main','ADMIN');await page.locator('.care-task').first().waitFor();
 const openHistory=async()=>{
  const requested=page.waitForRequest(r=>r.url().endsWith('/api/v1/care/events'));
  await page.getByRole('button',{name:'处置记录',exact:true}).first().click();return requested;
 };
 const historyRequest=await openHistory();
 const historyCanceled=page.waitForEvent('requestfailed',{predicate:r=>r===historyRequest});
 await page.getByRole('dialog',{name:'照护事件履历'}).locator('.el-dialog__headerbtn').click();await historyCanceled;
 check('Closing history aborts its hung XHR',historyRequests.length===1);
 const historyAgain=await openHistory();
 const historyUnmounted=page.waitForEvent('requestfailed',{predicate:r=>r===historyAgain});
 await page.evaluate(()=>{location.hash='/customer'});await historyUnmounted;
 await page.locator('.roles').waitFor();
 check('History can reopen and is aborted when leaving the page',historyRequests.length===2);
 await page.unroute('**/api/v1/care/events');await page.unroute('**/api/v1/care/incidents');

 await visit('/device','ADMIN');
 await page.getByRole('heading',{name:'设备管理',exact:true}).waitFor();
 await page.waitForTimeout(250);
 let release,started,requests=0;
 const gate=new Promise(resolve=>{release=resolve}),pending=new Promise(resolve=>{started=resolve});
 await page.route('**/api/v1/device/list',async route=>{requests++;started();await gate;await route.fulfill({contentType:'application/json',body:JSON.stringify({devices:[]})});});
 await Promise.race([pending,new Promise((_,reject)=>setTimeout(()=>reject(new Error('Refresh did not run')),12000))]);
 await page.locator('a[href="#/main"]').first().click();
 await page.getByRole('heading',{name:'运营总览',exact:true}).waitFor();
 release();await page.waitForTimeout(5500);
 check('Pending device refresh cannot restart polling after leaving the page',requests===1,{requests});
 await page.unroute('**/api/v1/device/list');
 for(const [route,role,endpoint,button,home,body]of[
  ['/ratepre','ADMIN','ratepre/selectRatepreDetail','查看健康详情','/main',{hrs:[72],bps:[120],createTimes:['08:00'],time:'2026-10-16'}],
  ['/step','ADMIN','step/selectStepDetail','查看睡眠详情','/main',{sleep:[7],createTime:['10-16']}],
  ['/rateprefamily','FAMILY','ratepre/selectRatepreDetail',/详情|详细|趋势/,'/familyMain',{hrs:[72],bps:[120],createTimes:['08:00'],time:'2026-10-16'}],
  ['/stepfamily','FAMILY','step/selectStepDetail',/详情|详细|趋势/,'/familyMain',{sleep:[7],createTime:['10-16']}]
 ]){
  await visit(route,role);await page.waitForTimeout(200);
  let resume,arrived;const hold=new Promise(r=>{resume=r}),request=new Promise(r=>{arrived=r});
  await page.route('**/api/v1/'+endpoint,async r=>{arrived();await hold;await r.fulfill({contentType:'application/json',body:JSON.stringify(body)});});
  await page.getByRole('button',{name:button}).first().click();await request;
  await page.evaluate(hash=>{location.hash=hash},home);
  await page.waitForURL(url=>url.hash==='#'+home);await page.locator('.el-dialog:visible').waitFor({state:'hidden'});
  resume();await page.waitForTimeout(350);
  check(route+': delayed chart result does not recreate a chart after navigation',await page.locator('[id^="myChart"] canvas').count()===0);
  await page.unroute('**/api/v1/'+endpoint);
 }
 check('No lifecycle JavaScript errors',report.pageErrors.length===0,report.pageErrors);
 check('No missing lifecycle assets',report.missingAssets.length===0,report.missingAssets);
 fs.writeFileSync(path.join(__dirname,'artifacts/lifecycle-acceptance.json'),JSON.stringify({checks},null,2));
 console.log(JSON.stringify({checks:checks.length,failed:checks.filter(c=>!c.pass)},null,2));if(checks.some(c=>!c.pass))process.exitCode=1;
}).catch(e=>{console.error(e);process.exitCode=1});

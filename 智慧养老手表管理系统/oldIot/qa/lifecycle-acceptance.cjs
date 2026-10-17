const fs=require('fs'),path=require('path'),run=require('./browser-harness.cjs');
run(async({page,base,report,setRole,setSignedIn})=>{
 const checks=[];const check=(name,pass,details)=>checks.push({name,pass:Boolean(pass),details});
 const visit=async(route,role)=>{setRole(role);setSignedIn(true);await page.goto(base+'/?lifecycle='+Date.now()+'#'+route);await page.locator('.care-app').waitFor();};
 await visit('/device','ADMIN');
 await page.getByRole('heading',{name:'设备管理',exact:true}).waitFor();
 await page.waitForTimeout(250);
 let release,started,requests=0;
 const gate=new Promise(resolve=>{release=resolve}),pending=new Promise(resolve=>{started=resolve});
 await page.route('**/restful/device/list',async route=>{requests++;started();await gate;await route.fulfill({contentType:'application/json',body:JSON.stringify({devices:[]})});});
 await Promise.race([pending,new Promise((_,reject)=>setTimeout(()=>reject(new Error('Refresh did not run')),12000))]);
 await page.locator('a[href="#/main"]').first().click();
 await page.getByRole('heading',{name:'运营总览',exact:true}).waitFor();
 release();await page.waitForTimeout(5500);
 check('Pending device refresh cannot restart polling after leaving the page',requests===1,{requests});
 await page.unroute('**/restful/device/list');
 for(const [route,role,endpoint,button,home,body]of[
  ['/ratepre','ADMIN','ratepre/selectRatepreDetail','查看健康详情','/main',{hrs:[72],bps:[120],createTimes:['08:00'],time:'2026-10-16'}],
  ['/step','ADMIN','step/selectStepDetail','查看睡眠详情','/main',{sleep:[7],createTime:['10-16']}],
  ['/rateprefamily','FAMILY','ratepre/selectRatepreDetail',/详情|详细|趋势/,'/familyMain',{hrs:[72],bps:[120],createTimes:['08:00'],time:'2026-10-16'}],
  ['/stepfamily','FAMILY','step/selectStepDetail',/详情|详细|趋势/,'/familyMain',{sleep:[7],createTime:['10-16']}]
 ]){
  await visit(route,role);await page.waitForTimeout(200);
  let resume,arrived;const hold=new Promise(r=>{resume=r}),request=new Promise(r=>{arrived=r});
  await page.route('**/restful/'+endpoint,async r=>{arrived();await hold;await r.fulfill({contentType:'application/json',body:JSON.stringify(body)});});
  await page.getByRole('button',{name:button}).first().click();await request;
  await page.evaluate(hash=>{location.hash=hash},home);
  await page.waitForURL(url=>url.hash==='#'+home);await page.locator('.el-dialog:visible').waitFor({state:'hidden'});
  resume();await page.waitForTimeout(350);
  check(route+': delayed chart result does not recreate a chart after navigation',await page.locator('[id^="myChart"] canvas').count()===0);
  await page.unroute('**/restful/'+endpoint);
 }
 check('No lifecycle JavaScript errors',report.pageErrors.length===0,report.pageErrors);
 check('No missing lifecycle assets',report.missingAssets.length===0,report.missingAssets);
 fs.writeFileSync(path.join(__dirname,'artifacts/lifecycle-acceptance.json'),JSON.stringify({checks},null,2));
 console.log(JSON.stringify({checks:checks.length,failed:checks.filter(c=>!c.pass)},null,2));if(checks.some(c=>!c.pass))process.exitCode=1;
}).catch(e=>{console.error(e);process.exitCode=1});

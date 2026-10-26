// Run ONLY against a new disposable backend/database with no real recipients.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const Module = require('node:module');
const base = process.env.WATCH_TEST_URL;
const bootstrap = process.env.WATCH_TEST_SETUP_TOKEN;
if (!/^http:\/\/127\.0\.0\.1:\d+$/.test(base || '') || !bootstrap) {
  throw new Error('需要专用本机测试后端 WATCH_TEST_URL 和 WATCH_TEST_SETUP_TOKEN；不要指向现有业务库');
}
const ts = require('typescript');
const source = path.resolve(__dirname, '../entry/src/main/ets/model/Protocol.ets');
const mod = new Module(source, module);
mod._compile(ts.transpileModule(fs.readFileSync(source, 'utf8'), {
  compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS }
}).outputText, source);
const protocol = mod.exports;
let cookie = '', csrf = '';
const checks = [];
async function request(route, body, headers = {}, expected = 200) {
  const response = await fetch(base + route, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', Cookie: cookie, 'X-CSRF-TOKEN': csrf, ...headers },
    body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(15000)
  });
  const setCookie = response.headers.get('set-cookie');
  if (setCookie) cookie = setCookie.split(';')[0];
  const result = await response.json();
  assert.equal(response.status, expected, route + ' status');
  if (expected === 200 && result.returnCode !== undefined) assert.equal(result.returnCode, '1111', route + ' rejected: ' + (result.message || result.returnCode));
  return result;
}
async function main() {
  assert.equal((await request('/restful/service/status')).databaseReady, true);
  const initial = await request('/restful/userLogin/session');
  assert.equal(initial.setupRequired, true, 'test must use an empty disposable database');
  csrf = initial.csrfToken;
  const password = 'Test' + crypto.randomBytes(8).toString('hex');
  await request('/restful/setup', { name: '模拟测试', phone: '13900000001', password }, { 'X-Suining-Desktop': bootstrap });
  await request('/restful/userLogin/login', { phone: '13900000001', password });
  csrf = (await request('/restful/userLogin/session')).csrfToken;
  await request('/restful/customer/add', { customerName: '模拟监护', customerAge: '40', customerSex: '男', customerPhone: '13900000002', customerAddress: '模拟器测试' });
  const customer = (await request('/restful/customer/select', { customerName: '模拟监护', page: 1, pageSize: 20 })).customers[0];
  await request('/restful/oldPerson/add', { name: '模拟长者', bothdate: '1950-01-01', sex: '男', phone: '13900000003', room: 'T01', illness: '', warn: '', foodLike: '', foodHate: '', drag: '', customerId: String(customer.customerId) });
  const elder = (await request('/restful/oldPerson/select', { name: '模拟长者', page: 1, pageSize: 20 })).oldPeople[0];
  const device = await request('/restful/device/create', { alias: 'WATCH5 模拟协议测试', model: 'WATCH5-SIM' });
  const credentials = { deviceNo: device.deviceNo, secret: device.secret };
  const before = await request('/restful/device/login', credentials);
  assert.equal(before.bound, 0);
  checks.push('new device authenticated and reported unbound');
  const code = await request('/restful/device/createBindCode', { oldId: String(elder.oldId), expireDays: '1' });
  await request('/restful/device/bind', { ...credentials, code: code.code });
  const config = protocol.normalizeConfig(base, device.deviceNo);
  const live = protocol.bindingFrom(config, await request('/restful/device/login', credentials));
  assert.equal(live.oldId, elder.oldId);
  checks.push('bind code attached device to test elder');
  assert.equal((await request('/restful/device/heartbeat', { ...credentials, appVersion: 'watch-sim-0.1.0' })).bound, 1);
  checks.push('heartbeat acknowledged');
  const samples = protocol.makeSamples(device.deviceNo, 'e2e-fixed-record', new Date(), 72, 1280);
  const queue = protocol.enqueue(undefined, live, samples);
  const body = protocol.batchRequest(queue, live, device.secret);
  assert.equal((await request('/restful/device/healthBatch', body)).accepted, 2);
  assert.equal((await request('/restful/device/healthBatch', body)).duplicates, 2);
  const changes = await request('/restful/health/changes', { oldId: elder.oldId, after: 0, limit: 100 });
  assert.equal(changes.changes.length, 2);
  checks.push('two simulated samples persisted; retry acknowledged two duplicates without extra changes');
  const sos = { ...credentials, requestId: 'watch-sim-e2e-same-press' };
  const first = await request('/restful/device/sos', sos);
  const retry = await request('/restful/device/sos', sos);
  assert.equal(retry.incidentId, first.incidentId);
  assert.equal(retry.duplicate, true);
  assert.equal(first.notificationState, 'no_recipient');
  checks.push('same SOS request ID reused one incident; no real notification recipient');
  await request('/restful/device/unbind', { deviceId: String(device.deviceId) });
  const code2 = await request('/restful/device/createBindCode', { oldId: String(elder.oldId), expireDays: '1' });
  await request('/restful/device/bind', { ...credentials, code: code2.code });
  const rebound = protocol.bindingFrom(config, await request('/restful/device/login', credentials));
  assert.notEqual(rebound.version, live.version);
  assert.throws(() => protocol.batchRequest(queue, rebound, device.secret));
  await request('/restful/device/healthBatch', body, {}, 409);
  checks.push('rebind changed epoch; both client and backend rejected old offline batch');
  console.log(JSON.stringify({ passed: checks.length, checks }, null, 2));
}
main().catch(error => { console.error(error.message); process.exitCode = 1; });

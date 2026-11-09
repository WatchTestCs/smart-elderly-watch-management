// Run ONLY against a new disposable backend/database with no real recipients.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const Module = require('node:module');
const http = require('node:http');
const os = require('node:os');
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
const load = require('./load-ets.cjs');
let transportProxy = '';
// Real HTTP to the disposable Go server through the production WatchApi.
const { WatchApi } = load('services/WatchApi', { '@kit.NetworkKit': { http: {
  RequestMethod: { POST: 'POST' }, HttpDataType: { STRING: 'STRING' },
  createHttp: () => ({ request: async (url, options) => {
    const response = await fetch(transportProxy ? transportProxy + new URL(url).pathname : url, { method: options.method, headers: options.header, body: options.extraData, signal: AbortSignal.timeout(options.readTimeout) });
    return { responseCode: response.status, result: await response.text() };
  }, destroy() {} })
} } });
const { NativeSampling } = load('model/NativeSampling');
const watch = new WatchApi();
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
  assert.equal((await request('/api/v1/service/status')).databaseReady, true);
  const initial = await request('/api/v1/auth/session');
  assert.equal(initial.setupRequired, true, 'test must use an empty disposable database');
  csrf = initial.csrfToken;
  const password = 'Test' + crypto.randomBytes(8).toString('hex');
  await request('/api/v1/setup', { name: '模拟测试', phone: '13900000001', password }, { 'X-Suining-Desktop': bootstrap });
  await request('/api/v1/auth/login', { phone: '13900000001', password });
  csrf = (await request('/api/v1/auth/session')).csrfToken;
  await request('/api/v1/guardians/create', { customerName: '模拟监护', customerAge: '40', customerSex: '男', customerPhone: '13900000002', customerAddress: '模拟器测试' });
  const customer = (await request('/api/v1/guardians/list', { customerName: '模拟监护', page: 1, pageSize: 20 })).customers[0];
  await request('/api/v1/residents/create', { name: '模拟长者', bothdate: '1950-01-01', sex: '男', phone: '13900000003', room: 'T01', illness: '', warn: '', foodLike: '', foodHate: '', drag: '', customerId: String(customer.customerId) });
  const elder = (await request('/api/v1/residents/list', { name: '模拟长者', page: 1, pageSize: 20 })).oldPeople[0];
  const device = await request('/api/v1/device/create', { alias: 'WATCH5 模拟协议测试', model: 'WATCH5-SIM' });
  const credentials = { deviceNo: device.deviceNo, secret: device.secret };
  const before = await request('/api/v1/device/login', credentials);
  assert.equal(before.bound, 0);
  checks.push('new device authenticated and reported unbound');
  const code = await request('/api/v1/device/createBindCode', { oldId: String(elder.oldId), expireDays: '1' });
  await request('/api/v1/device/bind', { ...credentials, code: code.code });
  const config = protocol.normalizeConfig(base, device.deviceNo);
  const live = protocol.bindingFrom(config, await watch.login(config, device.secret));
  assert.equal(live.oldId, elder.oldId);
  checks.push('bind code attached device to test elder');
  assert.equal((await watch.heartbeat(config, device.secret)).bound, 1);
  checks.push('heartbeat acknowledged');
  const samples = protocol.makeSamples(device.deviceNo, 'e2e-fixed-record', new Date(), 72, 1280);
  const queue = protocol.enqueue(undefined, live, samples);
  const body = protocol.batchRequest(queue, live, device.secret);
  assert.equal((await request('/api/v1/device/healthBatch', body)).accepted, 2);
  assert.equal((await request('/api/v1/device/healthBatch', body)).duplicates, 2);
  const changes = await request('/api/v1/health/changes', { oldId: elder.oldId, after: 0, limit: 100 });
  assert.equal(changes.changes.length, 2);
  checks.push('two simulated samples persisted; retry acknowledged two duplicates without extra changes');
  const native = new NativeSampling();
  const at = Date.now();
  assert.deepEqual(native.accept(live, 'native-anchor', { metric: 'steps', value: 100, receivedAt: at - 1000, sensorTimestamp: 1000000 }), []);
  const nativeSamples = native.accept(live, 'native-steps', { metric: 'steps', value: 105, receivedAt: at, sensorTimestamp: 2000000 })
    .concat(native.accept(live, 'native-hr', { metric: 'heart_rate', value: 74, receivedAt: at, sensorTimestamp: 2000000 }));
  const nativeQueue = protocol.enqueue(undefined, live, nativeSamples);
  // Real disk backing at the Preferences boundary, not a claim about device flash.
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'suining-watch-e2e-'));
  const prefsFile = path.join(temporary, 'preferences.json');
  const { WatchStore } = load('services/WatchStore', { '@kit.ArkData': { preferences: { getPreferencesSync: () => {
    const values = fs.existsSync(prefsFile) ? JSON.parse(fs.readFileSync(prefsFile, 'utf8')) : {};
    return { getSync: (key, fallback) => values[key] ?? fallback, putSync: (key, value) => { values[key] = value; },
      flush: async () => fs.writeFileSync(prefsFile, JSON.stringify(values)) };
  } } } });
  let forwarded = 0;
  let upstreamAck;
  const proxy = http.createServer(async (incoming, outgoing) => {
    try {
      assert.equal(incoming.url, '/api/v1/device/healthBatch');
      let payload = '';
      for await (const chunk of incoming) payload += chunk;
      forwarded++;
      const response = await fetch(base + incoming.url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: payload, signal: AbortSignal.timeout(10000) });
      assert.equal(response.status, 200);
      upstreamAck = await response.json();
      // The database committed, but the watch never receives its acknowledgement.
      outgoing.destroy();
    } catch (error) { outgoing.destroy(error); }
  });
  try {
    const store = new WatchStore({});
    await store.saveBatch(nativeQueue);
    await new Promise(resolve => proxy.listen(0, '127.0.0.1', resolve));
    transportProxy = 'http://127.0.0.1:' + proxy.address().port;
    await assert.rejects(watch.upload(config, protocol.batchRequest(store.batch(), live, device.secret)));
    transportProxy = '';
    assert.equal(forwarded, 1, 'a failed POST must not be replayed automatically');
    assert.equal(upstreamAck.accepted, 2, 'failure must occur after the real backend committed');
    const recovered = new WatchStore({});
    assert.deepEqual(recovered.batch(), nativeQueue, 'restart must retain the original IDs, values and binding');
    const retry = await watch.upload(config, protocol.batchRequest(recovered.batch(), live, device.secret));
    assert.equal(retry.accepted, 2);
    assert.equal(retry.duplicates, 2);
    await recovered.saveBatch(undefined);
    assert.equal(new WatchStore({}).batch(), undefined);
    checks.push('real socket loss after database commit retained the disk-backed queue; restart retry deduplicated original records before clearing');
  } finally {
    transportProxy = '';
    proxy.closeAllConnections();
    await new Promise(resolve => proxy.close(resolve));
    if (fs.existsSync(prefsFile)) fs.unlinkSync(prefsFile);
    fs.rmdirSync(temporary);
  }
  const latest = await request('/api/v1/health/latest', { oldId: elder.oldId });
  const nativeRecords = latest.items.filter(item => item.sample.source === 'suining-harmonyos');
  assert.equal(nativeRecords.length, 2);
  assert.equal(nativeRecords.find(item => item.sample.metric === 'steps').sample.value, 5);
  assert.ok(nativeRecords.every(item => item.sample.metadata.timestampBasis === 'app-received'));
  checks.push('production HarmonyOS transport negotiated, uploaded native-shaped heart/step samples, deduplicated retry and read latest with provenance');
  const sos = { ...credentials, requestId: 'watch-sim-e2e-same-press', bindingVersion: live.version };
  await request('/api/v1/device/sos', { ...credentials, requestId: 'missing-binding-version' }, {}, 400);
  await request('/api/v1/device/sos', { ...credentials, bindingVersion: live.version }, {}, 400);
  const first = await request('/api/v1/device/sos', sos);
  const retry = await request('/api/v1/device/sos', sos);
  assert.equal(retry.incidentId, first.incidentId);
  assert.equal(retry.duplicate, true);
  assert.equal(first.notificationState, 'no_recipient');
  checks.push('same SOS request ID reused one incident; no real notification recipient');
  await request('/api/v1/device/unbind', { deviceId: String(device.deviceId) });
  const code2 = await request('/api/v1/device/createBindCode', { oldId: String(elder.oldId), expireDays: '1' });
  await request('/api/v1/device/bind', { ...credentials, code: code2.code });
  const rebound = protocol.bindingFrom(config, await request('/api/v1/device/login', credentials));
  assert.notEqual(rebound.version, live.version);
  assert.throws(() => protocol.batchRequest(queue, rebound, device.secret));
  await request('/api/v1/device/healthBatch', body, {}, 409);
  await request('/api/v1/device/sos', sos, {}, 409);
  const currentSOS = await request('/api/v1/device/sos', { ...credentials, requestId: 'watch-sim-e2e-rebound', bindingVersion: rebound.version });
  assert.notEqual(currentSOS.incidentId, first.incidentId);
  checks.push('rebind changed epoch; both client and backend rejected old offline batch');
  checks.push('rebound device rejected old SOS retry and accepted a new SOS for the current binding');
  console.log(JSON.stringify({ passed: checks.length, checks }, null, 2));
}
main().catch(error => { console.error(error.message); process.exitCode = 1; });

const assert = require('node:assert/strict');
const test = require('node:test');
const fs = require('node:fs');
const path = require('node:path');
const Module = require('node:module');
const ts = require('typescript');
const source = path.resolve(__dirname, '../entry/src/main/ets/model/Protocol.ets');
const compiled = ts.transpileModule(fs.readFileSync(source, 'utf8'), {
  compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS }
}).outputText;
const mod = new Module(source, module);
mod._compile(compiled, source);
const p = mod.exports;
const config = p.normalizeConfig('http://127.0.0.1:18796/', 'watch5_sim_001');
const binding = p.bindingFrom(config, { returnCode: '1111', bound: 1, oldId: 7, bindingVersion: 3 });
const samples = p.makeSamples(config.deviceNo, 'stable-001', new Date('2026-10-25T04:00:00Z'), 72, 1280);

test('server validation rejects credentials, paths and query strings', () => {
  assert.equal(config.deviceNo, 'WATCH5_SIM_001');
  for (const url of ['file:///tmp', 'http://u:p@host', 'http://host/path', 'http://host?a=1', 'http://host:abc']) {
    assert.throws(() => p.normalizeConfig(url, 'WATCH1'));
  }
});
test('unbound or missing binding version cannot produce an authenticated owner', () => {
  assert.throws(() => p.bindingFrom(config, { returnCode: '1111', bound: 0 }));
  assert.throws(() => p.bindingFrom(config, { returnCode: '1111', bound: 1, oldId: 7 }));
});
test('samples declare simulation, units, and UTC cumulative day boundary', () => {
  assert.equal(samples[0].metadata.simulated, 'true');
  assert.equal(samples[0].start, samples[0].end);
  assert.equal(samples[0].unit, 'count/min');
  assert.equal(samples[1].start, '2026-10-25T00:00:00.000Z');
  assert.equal(samples[1].semantics, 'daily_total');
  assert.equal(samples[1].metadata.counterTimezone, 'UTC');
  assert.notEqual(samples[0].recordId, samples[1].recordId);
});
test('serialized offline batch retries preserve sample IDs and original binding version', () => {
  const queue = p.enqueue(undefined, binding, samples);
  const recovered = JSON.parse(JSON.stringify(queue));
  assert.deepEqual(p.batchRequest(queue, binding, 'secret'), p.batchRequest(recovered, binding, 'secret'));
  assert.equal(recovered.samples[0].recordId, 'stable-001-hr');
});
test('server, owner, device or epoch changes prevent uploading and mixing old samples', () => {
  const queue = p.enqueue(undefined, binding, samples);
  for (const change of [{ server: 'http://other' }, { oldId: 8 }, { deviceNo: 'WATCH2' }, { version: 4 }]) {
    const other = { ...binding, ...change };
    assert.throws(() => p.batchRequest(queue, other, 'secret'));
    assert.throws(() => p.enqueue(queue, other, samples));
  }
  assert.equal(queue.samples.length, 2);
});
test('queue capacity never silently evicts old records', () => {
  let queue;
  for (let i = 0; i < 100; i++) queue = p.enqueue(queue, binding, samples);
  assert.equal(queue.samples.length, 200);
  assert.throws(() => p.enqueue(queue, binding, samples));
  assert.equal(queue.samples.length, 200);
});

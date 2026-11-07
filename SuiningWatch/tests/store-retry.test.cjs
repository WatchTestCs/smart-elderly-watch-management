const test = require('node:test');
const assert = require('node:assert/strict');
const load = require('./load-ets.cjs');
const protocol = load('model/Protocol');

const config = protocol.normalizeConfig('http://127.0.0.1:18796', 'WATCH5_SIM_001');
const binding = protocol.bindingFrom(config, {
  returnCode: '1111', bound: 1, oldId: 7, bindingVersion: 3,
  protocolVersion: protocol.WATCH_PROTOCOL_VERSION, maxBatchSize: 200
});
const samples = protocol.makeSamples(binding.deviceNo, 'retry-fixed', new Date('2026-11-07T00:00:00Z'), 72, 100);

function makeStore(initial = {}) {
  const values = { ...initial };
  const prefs = {
    getSync: (key, fallback) => values[key] ?? fallback,
    putSync: (key, value) => { values[key] = value; },
    flush: async () => {}
  };
  const { WatchStore } = load('services/WatchStore', {
    '@kit.ArkData': { preferences: { getPreferencesSync: () => prefs } }
  });
  return { store: new WatchStore({}), values };
}

function loginFor(live = binding) {
  return {
    returnCode: '1111', bound: 1, oldId: live.oldId, bindingVersion: live.version,
    protocolVersion: protocol.WATCH_PROTOCOL_VERSION, maxBatchSize: 200
  };
}

test('reconnect keeps telemetry until the server acknowledges the complete durable batch', async () => {
  const batch = protocol.enqueue(undefined, binding, samples);
  const { store } = makeStore({ batch: JSON.stringify(batch) });
  const api = {
    login: async () => loginFor(),
    sos: async () => { throw new Error('unexpected SOS'); },
    upload: async () => ({ returnCode: '1111', accepted: 1 })
  };

  await assert.rejects(store.flushPending(api, config, 'x'.repeat(24)), /确认数量不匹配/);
  assert.deepEqual(store.batch(), batch);
});

test('reconnect never sends durable records across a changed binding epoch', async () => {
  const batch = protocol.enqueue(undefined, binding, samples);
  const pending = { binding, requestId: 'stable-old-sos' };
  const rebound = { ...binding, version: 4 };
  const { store } = makeStore({
    batch: JSON.stringify(batch),
    sos: JSON.stringify([pending])
  });
  let uploads = 0;
  let sosCalls = 0;
  const api = {
    login: async () => loginFor(rebound),
    upload: async () => { uploads++; return { returnCode: '1111', accepted: batch.samples.length }; },
    sos: async () => { sosCalls++; return { returnCode: '1111', incidentId: 99 }; }
  };

  const result = await store.flushPending(api, config, 'x'.repeat(24));
  assert.equal(result.telemetryFlushed, 0);
  assert.equal(result.sosFlushed, 0);
  assert.equal(uploads, 0);
  assert.equal(sosCalls, 0);
  assert.deepEqual(store.batch(), batch);
  assert.deepEqual(store.pendingSOS(), [pending]);
});

test('reconnect reuses the durable SOS request ID and persists its acknowledgement', async () => {
  const pending = { binding, requestId: 'stable-sos-request' };
  const { store } = makeStore({ sos: JSON.stringify([pending]) });
  let seen;
  const api = {
    login: async () => loginFor(),
    upload: async () => { throw new Error('unexpected upload'); },
    sos: async (_config, request) => {
      seen = request;
      return { returnCode: '1111', incidentId: 42 };
    }
  };

  const result = await store.flushPending(api, config, 'x'.repeat(24));
  assert.equal(result.sosFlushed, 1);
  assert.equal(seen.requestId, 'stable-sos-request');
  assert.equal(seen.bindingVersion, binding.version);
  assert.equal(store.pendingSOS()[0].incidentId, 42);
  assert.equal(protocol.pendingForBinding(store.pendingSOS(), binding), undefined);
});

test('reconnect clears telemetry only after an exact acknowledgement', async () => {
  const batch = protocol.enqueue(undefined, binding, samples);
  const { store } = makeStore({ batch: JSON.stringify(batch) });
  const api = {
    login: async () => loginFor(),
    sos: async () => { throw new Error('unexpected SOS'); },
    upload: async () => ({ returnCode: '1111', accepted: batch.samples.length, duplicates: batch.samples.length })
  };

  const result = await store.flushPending(api, config, 'x'.repeat(24));
  assert.equal(result.telemetryFlushed, batch.samples.length);
  assert.equal(store.batch(), undefined);
});

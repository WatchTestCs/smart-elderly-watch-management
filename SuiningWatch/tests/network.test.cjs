const test = require('node:test');
const assert = require('node:assert/strict');
const load = require('./load-ets.cjs');

function harness() {
  let available;
  let registerCalls = 0;
  let unregisterCalls = 0;
  const conn = {
    on(event, callback) {
      if (event === 'netAvailable') available = callback;
    },
    register(callback) {
      registerCalls++;
      callback(undefined);
    },
    unregister(callback) {
      unregisterCalls++;
      callback(undefined);
    }
  };
  const { WatchNetworkManager } = load('services/WatchNetworkManager', {
    '@kit.NetworkKit': { connection: { createNetConnection: () => conn } },
    '@kit.BasicServicesKit': {}
  });
  return { manager: new WatchNetworkManager(), fire: () => available?.({ netId: 1 }),
    registerCalls: () => registerCalls, unregisterCalls: () => unregisterCalls };
}

test('network restore retries are coalesced and lifecycle is idempotent', async () => {
  const h = harness();
  let release = () => {};
  let calls = 0;
  const callback = async () => {
    calls++;
    await new Promise(resolve => { release = resolve; });
  };

  h.manager.startMonitoring(callback);
  h.manager.startMonitoring(callback);
  assert.equal(h.registerCalls(), 1);

  h.fire();
  h.fire();
  await Promise.resolve();
  assert.equal(calls, 1, 'duplicate restore events are coalesced');

  release();
  await new Promise(resolve => setImmediate(resolve));

  h.fire();
  await Promise.resolve();
  assert.equal(calls, 2);

  release();
  await new Promise(resolve => setImmediate(resolve));
  h.manager.stopMonitoring();
  h.manager.stopMonitoring();
  assert.equal(h.unregisterCalls(), 1);
});

const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const load = require('./load-ets.cjs');
const { cryptoKit, arkTsKit } = require('./crypto-kit.cjs');

const fixedRandom = Buffer.from('00112233445566778899aabbccddeeff', 'hex');

test('device signer matches the backend HMAC contract byte-for-byte', async () => {
  const originalNow = Date.now;
  Date.now = () => 1_799_366_400_000;
  try {
    const { WatchRequestSigner } = load('services/WatchRequestSigner', {
      '@kit.CryptoArchitectureKit': cryptoKit(() => fixedRandom),
      '@kit.ArkTS': arkTsKit
    });
    const signer = new WatchRequestSigner();
    const body = JSON.stringify({
      deviceNo: 'WATCH001',
      secret: 'abcdefghijklmnopqrstuvwx',
      bindingVersion: 3,
      samples: [{ recordId: 'stable-1' }]
    });
    const result = await signer.sign('watch001', 'abcdefghijklmnopqrstuvwx', body);
    const timestamp = '1799366400';
    const nonce = fixedRandom.toString('hex');
    const bodyHash = crypto.createHash('sha256').update(body).digest('hex');
    const expected = crypto.createHmac('sha256', 'abcdefghijklmnopqrstuvwx')
      .update('WATCH001:' + timestamp + ':' + nonce + ':' + bodyHash)
      .digest('hex');

    assert.deepEqual(result, { signature: expected, timestamp, nonce });
  } finally {
    Date.now = originalNow;
  }
});

test('WatchApi signs every device request and sends the exact signed JSON bytes', async () => {
  const requests = [];
  const httpMock = {
    RequestMethod: { POST: 'POST' },
    HttpDataType: { STRING: 'STRING' },
    createHttp: () => ({
      request: async (url, options) => {
        requests.push({ url, options });
        return {
          responseCode: 200,
          result: JSON.stringify({
            returnCode: '1111', bound: 1, oldId: 7, bindingVersion: 3,
            protocolVersion: 1, maxBatchSize: 200, accepted: 1, incidentId: 42
          })
        };
      },
      destroy() {}
    })
  };
  const mocks = {
    '@kit.NetworkKit': { http: httpMock },
    '@kit.CryptoArchitectureKit': cryptoKit(() => fixedRandom),
    '@kit.ArkTS': arkTsKit
  };
  const { WatchApi } = load('services/WatchApi', mocks);
  const api = new WatchApi();
  const config = { server: 'https://watch.example', deviceNo: 'WATCH001' };
  const secret = 'abcdefghijklmnopqrstuvwx';

  await api.login(config, secret);
  await api.upload(config, {
    deviceNo: 'WATCH001', secret, bindingVersion: 3,
    samples: [{
      recordId: 'stable-1', source: 'harmonyos', sourceDevice: 'WATCH001',
      metric: 'heart_rate', unit: 'count/min', value: 72,
      start: '2026-11-07T00:00:00.000Z', end: '2026-11-07T00:00:00.000Z',
      semantics: 'instant', version: 1
    }]
  });
  await api.sos(config, { deviceNo: 'WATCH001', secret, requestId: 'stable-sos', bindingVersion: 3 });

  assert.equal(requests.length, 3);
  for (const request of requests) {
    assert.match(request.options.header['X-Device-Signature'], /^[0-9a-f]{64}$/);
    assert.match(request.options.header['X-Device-Nonce'], /^[0-9a-f]{32}$/);
    assert.match(request.options.header['X-Device-Timestamp'], /^\d+$/);
    const bodyHash = crypto.createHash('sha256').update(request.options.extraData).digest('hex');
    const expected = crypto.createHmac('sha256', secret)
      .update('WATCH001:' + request.options.header['X-Device-Timestamp'] + ':' +
        request.options.header['X-Device-Nonce'] + ':' + bodyHash)
      .digest('hex');
    assert.equal(request.options.header['X-Device-Signature'], expected);
  }
});

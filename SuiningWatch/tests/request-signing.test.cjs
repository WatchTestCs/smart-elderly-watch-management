const test = require('node:test');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const load = require('./load-ets.cjs');

const fixedRandom = Buffer.from('00112233445566778899aabbccddeeff', 'hex');

function cryptoMock() {
  return {
    cryptoFramework: {
      createMd: algorithm => {
        assert.equal(algorithm, 'SHA256');
        let data = Buffer.alloc(0);
        return {
          update: async blob => { data = Buffer.concat([data, Buffer.from(blob.data)]); },
          digest: async () => ({ data: new Uint8Array(crypto.createHash('sha256').update(data).digest()) })
        };
      },
      createRandom: () => ({
        generateRandom: async length => {
          assert.equal(length, 16);
          return { data: new Uint8Array(fixedRandom) };
        }
      }),
      createSymKeyGenerator: algorithm => {
        assert.equal(algorithm, 'HMAC');
        return { convertKey: async blob => ({ raw: Buffer.from(blob.data) }) };
      },
      createMac: algorithm => {
        assert.equal(algorithm, 'SHA256');
        let key;
        let data = Buffer.alloc(0);
        return {
          init: async symKey => { key = symKey.raw; },
          update: async blob => { data = Buffer.concat([data, Buffer.from(blob.data)]); },
          doFinal: async () => ({ data: new Uint8Array(crypto.createHmac('sha256', key).update(data).digest()) })
        };
      }
    }
  };
}

const arkTsMock = {
  buffer: {
    from(value, encoding) {
      return Buffer.from(value, encoding);
    }
  }
};

test('device signer matches the backend HMAC contract byte-for-byte', async () => {
  const originalNow = Date.now;
  Date.now = () => 1_799_366_400_000;
  try {
    const { WatchRequestSigner } = load('services/WatchRequestSigner', {
      '@kit.CryptoArchitectureKit': cryptoMock(),
      '@kit.ArkTS': arkTsMock
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

test('WatchApi signs only durable health and SOS writes and sends the exact signed JSON bytes', async () => {
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
    '@kit.CryptoArchitectureKit': cryptoMock(),
    '@kit.ArkTS': arkTsMock
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
  assert.equal(requests[0].options.header['X-Device-Signature'], undefined);
  for (const request of requests.slice(1)) {
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

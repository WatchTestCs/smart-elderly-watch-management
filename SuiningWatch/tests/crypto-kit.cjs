const crypto = require('node:crypto');

function cryptoKit(randomBytes = () => crypto.randomBytes(16)) {
  return {
    cryptoFramework: {
      createMd: algorithm => {
        if (algorithm !== 'SHA256') throw new Error('unexpected digest ' + algorithm);
        let data = Buffer.alloc(0);
        return {
          update: async blob => { data = Buffer.concat([data, Buffer.from(blob.data)]); },
          digest: async () => ({ data: new Uint8Array(crypto.createHash('sha256').update(data).digest()) })
        };
      },
      createRandom: () => ({
        generateRandom: async length => {
          if (length !== 16) throw new Error('unexpected nonce length ' + length);
          return { data: new Uint8Array(randomBytes()) };
        }
      }),
      createSymKeyGenerator: algorithm => {
        if (algorithm !== 'HMAC') throw new Error('unexpected key algorithm ' + algorithm);
        return { convertKey: async blob => ({ raw: Buffer.from(blob.data) }) };
      },
      createMac: algorithm => {
        if (algorithm !== 'SHA256') throw new Error('unexpected MAC ' + algorithm);
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

const arkTsKit = {
  buffer: {
    from(value, encoding) {
      return Buffer.from(value, encoding);
    }
  }
};

module.exports = { cryptoKit, arkTsKit };

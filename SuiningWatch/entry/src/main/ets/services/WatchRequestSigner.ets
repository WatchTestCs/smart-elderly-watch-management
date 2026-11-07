import { cryptoFramework } from '@kit.CryptoArchitectureKit';
import { buffer } from '@kit.ArkTS';

export interface DeviceSignature {
  signature: string;
  timestamp: string;
  nonce: string;
}

function bytes(value: string): Uint8Array {
  return new Uint8Array(buffer.from(value, 'utf-8'));
}

function hex(value: Uint8Array): string {
  return buffer.from(value).toString('hex');
}

export class WatchRequestSigner {
  async sign(deviceNo: string, secret: string, body: string): Promise<DeviceSignature> {
    const md = cryptoFramework.createMd('SHA256');
    await md.update({ data: bytes(body) });
    const bodyDigest = await md.digest();

    const random = cryptoFramework.createRandom();
    const nonceData = await random.generateRandom(16);
    const nonce = hex(nonceData.data);
    const timestamp = Math.floor(Date.now() / 1000).toString();

    const keyGenerator = cryptoFramework.createSymKeyGenerator('HMAC');
    const key = await keyGenerator.convertKey({ data: bytes(secret) });
    const mac = cryptoFramework.createMac('SHA256');
    await mac.init(key);
    await mac.update({
      data: bytes(deviceNo.toUpperCase() + ':' + timestamp + ':' + nonce + ':' + hex(bodyDigest.data))
    });
    const result = await mac.doFinal();
    return { signature: hex(result.data), timestamp: timestamp, nonce: nonce };
  }
}

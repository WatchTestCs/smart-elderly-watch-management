import { http } from '@kit.NetworkKit';
import { WatchConfig, Credentials, DeviceReply, BindRequest, HeartbeatRequest,
  BatchRequest, SOSRequest, LoginRequest, DeviceRequestError, WATCH_APP_VERSION, WATCH_PROTOCOL_VERSION, verifyHandshake } from '../model/Protocol';

// A single request has a bounded lifetime; requests never log the device secret.
export class WatchApi {
  private async post(config: WatchConfig, path: string,
    body: Credentials | LoginRequest | BindRequest | HeartbeatRequest | BatchRequest | SOSRequest): Promise<DeviceReply> {
    const request = http.createHttp();
    try {
      const response = await request.request(config.server + '/api/v1/device/' + path, {
        method: http.RequestMethod.POST,
        header: { 'Content-Type': 'application/json' },
        extraData: JSON.stringify(body),
        usingCache: false,
        expectDataType: http.HttpDataType.STRING,
        connectTimeout: 5000, readTimeout: 10000
      });
      const reply = JSON.parse(response.result as string) as DeviceReply;
      if (response.responseCode < 200 || response.responseCode >= 300 || reply.returnCode !== '1111') {
        const message = reply.message || '服务返回 HTTP ' + response.responseCode;
        throw new DeviceRequestError(reply.code || 'HTTP_ERROR', message + (reply.reference ? '（编号：' + reply.reference + '）' : ''));
      }
      return reply;
    } finally {
      request.destroy();
    }
  }
  async login(config: WatchConfig, secret: string): Promise<DeviceReply> {
    const reply = await this.post(config, 'login', { deviceNo: config.deviceNo, secret: secret,
      platform: 'harmonyos', protocolVersion: WATCH_PROTOCOL_VERSION });
    verifyHandshake(reply);
    return reply;
  }
  bind(config: WatchConfig, secret: string, code: string): Promise<DeviceReply> {
    return this.post(config, 'bind', { deviceNo: config.deviceNo, secret: secret, code: code.trim().toUpperCase() });
  }
  heartbeat(config: WatchConfig, secret: string): Promise<DeviceReply> {
    return this.post(config, 'heartbeat', { deviceNo: config.deviceNo, secret: secret, appVersion: WATCH_APP_VERSION });
  }
  upload(config: WatchConfig, body: BatchRequest): Promise<DeviceReply> {
    return this.post(config, 'healthBatch', body);
  }
  sos(config: WatchConfig, body: SOSRequest): Promise<DeviceReply> {
    return this.post(config, 'sos', body);
  }
}

import { http } from '@kit.NetworkKit';
import { WatchConfig, Credentials, DeviceReply, BindRequest, HeartbeatRequest,
  BatchRequest, SOSRequest } from '../model/Protocol';

// A single request has a bounded lifetime; requests never log the device secret.
export class WatchApi {
  private async post(config: WatchConfig, path: string,
    body: Credentials | BindRequest | HeartbeatRequest | BatchRequest | SOSRequest): Promise<DeviceReply> {
    const request = http.createHttp();
    try {
      const response = await request.request(config.server + '/api/v1/device/' + path, {
        method: http.RequestMethod.POST,
        header: { 'Content-Type': 'application/json' },
        extraData: JSON.stringify(body),
        expectDataType: http.HttpDataType.STRING,
        connectTimeout: 5000, readTimeout: 10000
      });
      if (response.responseCode < 200 || response.responseCode >= 300) {
        if (response.responseCode === 409) {
          throw new Error('绑定版本冲突，请重新认证；旧记录已保留');
        }
        throw new Error('服务返回 HTTP ' + response.responseCode + '，请检查服务状态');
      }
      const reply = JSON.parse(response.result as string) as DeviceReply;
      if (reply.returnCode !== '1111') {
        throw new Error(reply.message || '设备请求未被接受，请检查设备号、密钥及绑定');
      }
      return reply;
    } finally {
      request.destroy();
    }
  }
  login(config: WatchConfig, secret: string): Promise<DeviceReply> {
    return this.post(config, 'login', { deviceNo: config.deviceNo, secret: secret });
  }
  bind(config: WatchConfig, secret: string, code: string): Promise<DeviceReply> {
    return this.post(config, 'bind', { deviceNo: config.deviceNo, secret: secret, code: code.trim().toUpperCase() });
  }
  heartbeat(config: WatchConfig, secret: string): Promise<DeviceReply> {
    return this.post(config, 'heartbeat', { deviceNo: config.deviceNo, secret: secret, appVersion: 'watch-sim-0.1.0' });
  }
  upload(config: WatchConfig, body: BatchRequest): Promise<DeviceReply> {
    return this.post(config, 'healthBatch', body);
  }
  sos(config: WatchConfig, body: SOSRequest): Promise<DeviceReply> {
    return this.post(config, 'sos', body);
  }
}

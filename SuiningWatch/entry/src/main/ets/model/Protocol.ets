// Platform-independent protocol logic. Sample IDs and binding identity survive retries.
export interface WatchConfig {
  server: string;
  deviceNo: string;
}
export interface SimulatorState { day: string; steps: number; }
export interface DeviceReply {
  returnCode: string;
  message?: string;
  deviceId?: number;
  bound?: number;
  oldId?: number;
  oldPersonName?: string;
  bindingVersion?: number;
  accepted?: number;
  duplicates?: number;
  incidentId?: number;
  incidentState?: string;
  notificationState?: string;
}
export interface Credentials { deviceNo: string; secret: string; }
export interface Binding {
  server: string;
  deviceNo: string;
  oldId: number;
  version: number;
}
export interface SampleMetadata {
  simulated: string;
  counterTimezone?: string;
}
export interface HealthSample {
  recordId: string;
  source: string;
  sourceDevice: string;
  sourceVersion: string;
  metric: string;
  unit: string;
  value: number;
  start: string;
  end: string;
  semantics: string;
  metadata: SampleMetadata;
  version: number;
}
export interface PendingBatch { binding: Binding; samples: HealthSample[]; }
export interface BatchRequest {
  deviceNo: string;
  secret: string;
  bindingVersion: number;
  samples: HealthSample[];
}
export interface PendingSOS { binding: Binding; requestId: string; incidentId?: number; }
export interface SOSRequest extends Credentials { requestId: string; bindingVersion: number; }
export function readPendingSOS(raw: string): PendingSOS[] {
  if (!raw) { return []; }
  const saved = JSON.parse(raw) as PendingSOS | PendingSOS[];
  return Array.isArray(saved) ? saved : [saved];
}
export function pendingForBinding(items: PendingSOS[], binding: Binding): PendingSOS | undefined {
  return items.find((item: PendingSOS) => !item.incidentId && sameBinding(item.binding, binding));
}
export interface HeartbeatRequest extends Credentials { appVersion: string; }
export interface BindRequest extends Credentials { code: string; }

export function normalizeConfig(server: string, deviceNo: string): WatchConfig {
  const url = server.trim().replace(/\/+$/, '');
  if (!/^https?:\/\/(?:\[[0-9a-fA-F:]+\]|[A-Za-z0-9.-]+)(?::[0-9]{1,5})?$/.test(url)) {
    throw new Error('服务器地址须为 http(s)://主机:端口，不含路径或账号');
  }
  const no = deviceNo.trim().toUpperCase();
  if (!/^[A-Z0-9_-]{1,32}$/.test(no)) {
    throw new Error('设备号须为 1–32 位字母、数字、下划线或短横线');
  }
  return { server: url, deviceNo: no };
}

export function bindingFrom(config: WatchConfig, reply: DeviceReply): Binding {
  if (reply.bound !== 1 || !reply.oldId || !reply.bindingVersion) {
    throw new Error('设备尚未绑定长者，请输入后台生成的设备绑定码');
  }
  return { server: config.server, deviceNo: config.deviceNo,
    oldId: reply.oldId, version: reply.bindingVersion };
}
export function sameBinding(a: Binding, b: Binding): boolean {
  return a.server === b.server && a.deviceNo === b.deviceNo && a.oldId === b.oldId && a.version === b.version;
}
export function enqueue(current: PendingBatch | undefined, binding: Binding,
  samples: HealthSample[]): PendingBatch {
  if (current && !sameBinding(current.binding, binding)) {
    throw new Error('待上传记录属于旧绑定，请先处理旧记录；不会归入新长者');
  }
  const prior: HealthSample[] = current ? current.samples : [];
  if (prior.length + samples.length > 200) {
    throw new Error('待上传记录已满，请先上传；本次采样未入队');
  }
  return { binding: binding, samples: prior.concat(samples) };
}
export function batchRequest(batch: PendingBatch, live: Binding, secret: string): BatchRequest {
  if (!sameBinding(batch.binding, live)) {
    throw new Error('服务器或设备绑定已变化，禁止补传旧记录');
  }
  return { deviceNo: live.deviceNo, secret: secret,
    bindingVersion: batch.binding.version, samples: batch.samples };
}
export function makeSamples(deviceNo: string, recordId: string, at: Date,
  heart: number, steps: number): HealthSample[] {
  const instant = at.toISOString();
  // Counter resets at a UTC day boundary; declare the same day basis to the server.
  const dayStart = new Date(Date.UTC(at.getUTCFullYear(), at.getUTCMonth(), at.getUTCDate())).toISOString();
  const heartSample: HealthSample = {
    recordId: recordId + '-hr', source: 'suining-watch-sim', sourceDevice: deviceNo,
    sourceVersion: '0.1.0', metric: 'heart_rate', unit: 'count/min', value: heart,
    start: instant, end: instant, semantics: 'instant', version: 1,
    metadata: { simulated: 'true' }
  };
  const stepSample: HealthSample = {
    recordId: recordId + '-steps', source: 'suining-watch-sim', sourceDevice: deviceNo,
    sourceVersion: '0.1.0', metric: 'steps', unit: 'count', value: steps,
    start: dayStart, end: instant, semantics: 'daily_total', version: 1,
    metadata: { simulated: 'true', counterTimezone: 'UTC' }
  };
  return [heartSample, stepSample];
}

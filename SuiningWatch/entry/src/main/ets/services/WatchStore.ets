import { preferences } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { WatchConfig, PendingBatch, PendingSOS, SimulatorState, DeviceReply, BatchRequest, SOSRequest,
  Binding, readPendingSOS, bindingFrom, sameBinding, batchRequest } from '../model/Protocol';

export interface WatchApiDispatcher {
  login(config: WatchConfig, secret: string): Promise<DeviceReply>;
  upload(config: WatchConfig, body: BatchRequest): Promise<DeviceReply>;
  sos(config: WatchConfig, body: SOSRequest): Promise<DeviceReply>;
}

export interface FlushResult {
  telemetryFlushed: number;
  sosFlushed: number;
  binding?: Binding;
}

export class WatchStore {
  private prefs: preferences.Preferences;
  private sosStorageFailed: boolean = false;
  private batchStorageFailed: boolean = false;
  constructor(context: common.Context) {
    this.prefs = preferences.getPreferencesSync(context, { name: 'suining_watch_sim_v1' });
  }
  config(): WatchConfig {
    return {
      server: this.prefs.getSync('server', 'http://127.0.0.1:18796') as string,
      deviceNo: this.prefs.getSync('deviceNo', 'WATCH5_SIM_001') as string
    };
  }
  async saveConfig(config: WatchConfig): Promise<void> {
    this.prefs.putSync('server', config.server);
    this.prefs.putSync('deviceNo', config.deviceNo);
    await this.prefs.flush();
  }
  batch(): PendingBatch | undefined {
    if (this.batchStorageFailed) { throw new Error('采样保存状态未确认，请重启应用恢复记录，勿重复采样'); }
    const raw = this.prefs.getSync('batch', '') as string;
    return raw ? JSON.parse(raw) as PendingBatch : undefined;
  }
  pendingSOS(): PendingSOS[] {
    if (this.sosStorageFailed) { throw new Error('求助记录保存失败，请联系管理员核查，勿重复发送'); }
    const raw = this.prefs.getSync('sos', '') as string;
    return readPendingSOS(raw);
  }
  async saveBatch(batch: PendingBatch | undefined): Promise<void> {
    if (this.batchStorageFailed) { throw new Error('采样保存状态未确认，请重启应用恢复记录'); }
    try {
      this.prefs.putSync('batch', batch ? JSON.stringify(batch) : '');
      await this.prefs.flush();
    } catch (error) {
      this.batchStorageFailed = true;
      throw new Error('采样保存状态未确认，请重启应用恢复记录，勿重复采样');
    }
  }
  async saveSOS(sos: PendingSOS[]): Promise<void> {
    if (this.sosStorageFailed) { throw new Error('求助记录保存失败，请联系管理员核查，勿重复发送'); }
    const previous = this.prefs.getSync('sos', '') as string;
    try {
      this.prefs.putSync('sos', sos.length ? JSON.stringify(sos) : '');
      await this.prefs.flush();
    } catch (error) {
      this.sosStorageFailed = true;
      try { this.prefs.putSync('sos', previous); } catch (restoreError) { /* Keep sending disabled. */ }
      throw new Error('求助记录保存失败，请联系管理员核查，勿重复发送');
    }
  }
  async flushPending(api: WatchApiDispatcher, config: WatchConfig, secret: string): Promise<FlushResult> {
    const batch = this.batch();
    let items = this.pendingSOS();
    const hasPendingSOS = items.some((item) => !item.incidentId);
    if (!batch && !hasPendingSOS) {
      return { telemetryFlushed: 0, sosFlushed: 0 };
    }

    const login = await api.login(config, secret);
    if (login.bound !== 1) {
      return { telemetryFlushed: 0, sosFlushed: 0 };
    }
    const live = bindingFrom(config, login);
    let sosFlushed = 0;

    for (const pending of items.filter((item) => !item.incidentId && sameBinding(item.binding, live))) {
      const reply = await api.sos(config, {
        deviceNo: live.deviceNo,
        secret: secret,
        requestId: pending.requestId,
        bindingVersion: pending.binding.version
      });
      if (!reply.incidentId) {
        throw new Error('后台确认缺少求助编号，原请求已保留');
      }
      const incidentId = reply.incidentId;
      items = items.map((item) => item.requestId === pending.requestId && sameBinding(item.binding, live)
        ? { binding: item.binding, requestId: item.requestId, incidentId: incidentId } : item);
      // Persist each acknowledgement before moving to the next request. If a
      // later retry fails, already-confirmed SOS entries stay confirmed.
      await this.saveSOS(items);
      sosFlushed++;
    }

    let telemetryFlushed = 0;
    const pendingBatch = this.batch();
    if (pendingBatch && sameBinding(pendingBatch.binding, live)) {
      const reply = await api.upload(config, batchRequest(pendingBatch, live, secret));
      // A generic success code is insufficient: only an exact acknowledgement
      // proves every durable sample can be removed locally.
      if (reply.accepted !== pendingBatch.samples.length) {
        throw new Error('上传确认数量不匹配，记录已保留');
      }
      await this.saveBatch(undefined);
      telemetryFlushed = pendingBatch.samples.length;
    }

    return { telemetryFlushed: telemetryFlushed, sosFlushed: sosFlushed, binding: live };
  }

  simulator(): SimulatorState {
    return { day: this.prefs.getSync('counterDay', '') as string,
      steps: this.prefs.getSync('counterSteps', 1280) as number };
  }
  async saveSimulator(state: SimulatorState): Promise<void> {
    this.prefs.putSync('counterDay', state.day);
    this.prefs.putSync('counterSteps', state.steps);
    await this.prefs.flush();
  }
  nextID(): string {
    // Monotonic persistent sequence prevents collisions across restart / clock rollback.
    const sequence = (this.prefs.getSync('sequence', 0) as number) + 1;
    this.prefs.putSync('sequence', sequence);
    return Date.now().toString(36) + '-' + sequence.toString(36);
  }
}

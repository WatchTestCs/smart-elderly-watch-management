import { preferences } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { WatchConfig, PendingBatch, PendingSOS, SimulatorState, readPendingSOS } from '../model/Protocol';

export class WatchStore {
  private prefs: preferences.Preferences;
  private sosStorageFailed: boolean = false;
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
    const raw = this.prefs.getSync('batch', '') as string;
    return raw ? JSON.parse(raw) as PendingBatch : undefined;
  }
  pendingSOS(): PendingSOS[] {
    if (this.sosStorageFailed) { throw new Error('求助记录保存失败，请联系管理员核查，勿重复发送'); }
    const raw = this.prefs.getSync('sos', '') as string;
    return readPendingSOS(raw);
  }
  async saveBatch(batch: PendingBatch | undefined): Promise<void> {
    this.prefs.putSync('batch', batch ? JSON.stringify(batch) : '');
    await this.prefs.flush();
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

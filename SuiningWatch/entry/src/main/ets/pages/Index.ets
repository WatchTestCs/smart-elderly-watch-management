import { common } from '@kit.AbilityKit';
import { WatchApi } from '../services/WatchApi';
import { WatchStore } from '../services/WatchStore';
import { Binding, WatchConfig,
  normalizeConfig, bindingFrom, enqueue, batchRequest, makeSamples, sameBinding } from '../model/Protocol';

@Entry
@Component
struct Index {
  @State private page: number = 0;
  @State private server: string = 'http://127.0.0.1:18796';
  @State private deviceNo: string = 'WATCH5_SIM_001';
  @State private secret: string = '';
  @State private bindCode: string = '';
  @State private person: string = '尚未绑定';
  @State private connected: boolean = false;
  @State private heart: number = 72;
  @State private steps: number = 1280;
  @State private queued: number = 0;
  @State private busy: boolean = false;
  @State private status: string = '本地预览 · 尚未连接后台';
  @State private lastUpload: string = '尚未上传';
  @State private sosPending: boolean = false;
  private api: WatchApi = new WatchApi();
  private store: WatchStore | undefined = undefined;
  private binding: Binding | undefined = undefined;
  private counterDay: string = new Date().toISOString().slice(0, 10);

  aboutToAppear(): void {
    try {
      const context = this.getUIContext().getHostContext();
      if (!context) { throw new Error('无法取得应用上下文'); }
      this.store = new WatchStore(context as common.UIAbilityContext);
      const config = this.store.config();
      this.server = config.server;
      this.deviceNo = config.deviceNo;
      const batch = this.store.batch();
      this.queued = batch ? batch.samples.length : 0;
      this.sosPending = this.store.pendingSOS() !== undefined;
      const simulator = this.store.simulator();
      if (simulator.day === this.counterDay) { this.steps = simulator.steps; }
    } catch (error) {
      this.status = '读取本地状态失败，请重新打开应用；不会覆盖旧记录';
    }
  }
  private config(): WatchConfig { return normalizeConfig(this.server, this.deviceNo); }
  private credentials(): void {
    if (this.secret.length < 24 || this.secret.length > 128) {
      throw new Error('请输入后台生成的设备密钥（24–128 位），不是管理员密码');
    }
    if (!this.store) { throw new Error('本地存储未就绪，请重新打开应用'); }
  }
  private async liveBinding(): Promise<Binding> {
    this.credentials();
    this.connected = false;
    const config = this.config();
    const reply = await this.api.login(config, this.secret);
    const binding = bindingFrom(config, reply);
    this.person = reply.oldPersonName || '长者 #' + binding.oldId;
    this.binding = binding;
    this.connected = true;
    return binding;
  }
  private async act(action: () => Promise<void>): Promise<void> {
    if (this.busy) { return; }
    this.busy = true;
    try { await action(); }
    catch (error) {
      const message = (error as Error).message;
      this.status = message || '请求失败，请检查连接；待上传记录已保留';
      this.connected = false;
    } finally { this.busy = false; }
  }
  private async connect(): Promise<void> {
    this.credentials();
    const config = this.config();
    await this.store?.saveConfig(config);
    const reply = await this.api.login(config, this.secret);
    if (reply.bound !== 1) {
      this.connected = false;
      this.binding = undefined;
      this.person = '尚未绑定';
      this.status = '设备认证成功，请输入设备绑定码';
      return;
    }
    this.binding = bindingFrom(config, reply);
    this.connected = true;
    this.person = reply.oldPersonName || '长者 #' + this.binding.oldId;
    this.status = '已认证 · ' + this.person;
  }
  private async bind(): Promise<void> {
    this.credentials();
    if (!/^[A-Za-z0-9]{4,20}$/.test(this.bindCode.trim())) { throw new Error('请输入有效的设备绑定码'); }
    await this.api.bind(this.config(), this.secret, this.bindCode);
    this.bindCode = '';
    await this.connect();
  }
  private async sample(): Promise<void> {
    const day = new Date().toISOString().slice(0, 10);
    if (day !== this.counterDay) { this.steps = 0; this.counterDay = day; }
    this.heart = 68 + Math.floor(Math.random() * 13);
    this.steps += 8 + Math.floor(Math.random() * 15);
    await this.store?.saveSimulator({ day: this.counterDay, steps: this.steps });
    if (!this.binding || !this.store) {
      this.status = '仅本地模拟，认证并绑定后才能保存上传记录';
      return;
    }
    const batch = enqueue(this.store.batch(), this.binding,
      makeSamples(this.binding.deviceNo, this.store.nextID(), new Date(), this.heart, this.steps));
    await this.store.saveBatch(batch);
    this.queued = batch.samples.length;
    this.status = '已保存 2 条模拟采样，点击“上传记录”提交';
  }
  private async upload(): Promise<void> {
    if (!this.store) { throw new Error('本地存储未就绪'); }
    const batch = this.store.batch();
    if (!batch) { throw new Error('当前没有待上传记录'); }
    const live = await this.liveBinding();
    const reply = await this.api.upload(this.config(), batchRequest(batch, live, this.secret));
    // Clear only after the server acknowledges the complete batch.
    if (reply.accepted !== batch.samples.length) { throw new Error('上传确认数量不匹配，记录已保留'); }
    await this.store.saveBatch(undefined);
    this.queued = 0;
    this.lastUpload = new Date().toLocaleTimeString();
    this.status = '后台已接收 ' + reply.accepted + ' 条，重复 ' + (reply.duplicates || 0) + ' 条';
  }
  private confirmSOS(): void {
    this.getUIContext().showAlertDialog({
      title: this.sosPending ? '重试同一次模拟求助？' : '发送模拟求助？',
      message: '将在真实后台生成测试照护任务。接收成功不代表通知送达或人员到场。',
      primaryButton: { value: '取消', action: () => {} },
      secondaryButton: { value: '确认发送', action: () => { this.act(async () => { await this.sendSOS(); }); } }
    });
  }
  private async sendSOS(): Promise<void> {
    if (!this.store) { throw new Error('本地存储未就绪'); }
    const live = await this.liveBinding();
    let pending = this.store.pendingSOS();
    if (pending && !sameBinding(pending.binding, live)) {
      throw new Error('旧求助与当前绑定不同，禁止发送；请联系管理员核查');
    }
    if (!pending) {
      pending = { binding: live, requestId: 'watch-sim-' + this.store.nextID() };
      await this.store.saveSOS(pending);
      this.sosPending = true;
    }
    const reply = await this.api.sos(this.config(), { deviceNo: live.deviceNo,
      secret: this.secret, requestId: pending.requestId });
    await this.store.saveSOS(undefined);
    this.sosPending = false;
    this.status = '求助 #' + (reply.incidentId || '') + ' 已接收；通知状态：' + (reply.notificationState || '待确认');
  }
  private confirmDiscard(): void {
    this.getUIContext().showAlertDialog({ title: '丢弃待上传的模拟记录？',
      message: '只清除本机模拟采样，不删除后台数据，也不清除未确认的求助。',
      primaryButton: { value: '保留', action: () => {} },
      secondaryButton: { value: '丢弃', action: () => {
        this.act(async () => { await this.store?.saveBatch(undefined); this.queued = 0; this.status = '本机模拟记录已丢弃'; });
      } }
    });
  }
  @Builder
  private action(label: string, callback: () => void, color: string = '#24463C') {
    Button(label).fontSize(15).fontColor('#F3EDE0').backgroundColor(color)
      .width('100%').height(40).enabled(!this.busy).onClick(callback)
  }
  @Builder
  private caption(label: string) {
    Text(label).fontSize(12).fontColor('#93B1A7').width('100%')
  }
  @Builder
  private home() {
    Text(this.person === '尚未绑定' ? '今天，也安心' : this.person).fontSize(19).fontColor('#F3EDE0')
    Text('心率 · 模拟').fontSize(12).fontColor('#93B1A7')
    Row({ space: 4 }) {
      Text(this.heart.toString()).fontSize(52).fontWeight(FontWeight.Medium).fontColor('#E6CE9D')
      Text('次/分').fontSize(12).fontColor('#93B1A7').padding({ top: 22 })
    }.justifyContent(FlexAlign.Center)
    Text('步数 ' + this.steps + '  ·  模拟').fontSize(15).fontColor('#D8E4DC')
    this.action('生成模拟采样', () => { this.act(async () => { await this.sample(); }); })
    this.action(this.sosPending ? '重试模拟求助' : 'SOS 模拟求助', () => { this.confirmSOS(); }, '#803F37')
  }
  @Builder
  private records() {
    Text('数据同步').fontSize(21).fontColor('#F3EDE0')
    Text(this.queued.toString()).fontSize(42).fontColor('#E6CE9D')
    Text('条模拟采样待上传').fontSize(13).fontColor('#93B1A7')
    this.caption('最近上传：' + this.lastUpload)
    this.action('上传记录', () => { this.act(async () => { await this.upload(); }); })
    this.action('发送心跳', () => { this.act(async () => {
      this.credentials();
      const reply = await this.api.heartbeat(this.config(), this.secret);
      this.connected = false;
      this.binding = undefined;
      if (reply.bound === 1) { this.binding = bindingFrom(this.config(), reply); this.connected = true; }
      this.status = '心跳成功 · 当前绑定已校验';
    }); })
    this.action('丢弃本机模拟记录', () => { this.confirmDiscard(); }, '#373B35')
    this.caption('失败不清除队列；重试复用记录 ID，后台去重。绑定变化时禁止补传。')
  }
  @Builder
  private settings() {
    Text('连接与绑定').fontSize(21).fontColor('#F3EDE0')
    this.caption('后端地址')
    TextInput({ text: this.server, placeholder: 'http://127.0.0.1:18796' }).fontSize(12)
      .height(38).enabled(!this.busy).onChange((value: string) => { this.server = value; this.connected = false; this.binding = undefined; })
    this.caption('设备号（后台登记）')
    TextInput({ text: this.deviceNo, placeholder: 'WATCH5_SIM_001' }).fontSize(13)
      .height(38).enabled(!this.busy).onChange((value: string) => { this.deviceNo = value; this.connected = false; this.binding = undefined; })
    this.caption('设备密钥（仅本次运行使用）')
    TextInput({ text: this.secret, placeholder: '后台生成的设备密钥' }).type(InputType.Password).fontSize(13)
      .height(38).enabled(!this.busy).onChange((value: string) => { this.secret = value; this.connected = false; this.binding = undefined; })
    this.action('保存地址并认证', () => { this.act(async () => { await this.connect(); }); })
    this.caption('设备绑定码（不是家属邀请码）')
    TextInput({ text: this.bindCode, placeholder: '输入设备绑定码' }).fontSize(15)
      .height(38).enabled(!this.busy).onChange((value: string) => { this.bindCode = value; })
    this.action('绑定长者', () => { this.act(async () => { await this.bind(); }); })
    this.caption('本版仅模拟器采样，不读取真实心率、血压或睡眠。')
  }
  build() {
    Scroll() {
      Column({ space: 9 }) {
        Text('岁宁守护').fontSize(15).fontColor('#D5BD89').fontWeight(FontWeight.Medium)
        Text('模拟器演示 · 非真实健康数据').fontSize(10).fontColor('#D5BD89')
        Row({ space: 7 }) {
          ForEach(['首页', '记录', '设置'], (label: string, index: number) => {
            Button(label).fontSize(12).height(28).backgroundColor(this.page === index ? '#3D5F50' : '#172D26')
              .onClick(() => { this.page = index; })
          }, (label: string) => label)
        }.justifyContent(FlexAlign.Center)
        if (this.page === 0) { this.home(); }
        else if (this.page === 1) { this.records(); }
        else { this.settings(); }
        Text(this.busy ? '正在处理，请稍候…' : this.status).fontSize(12)
          .fontColor(this.connected ? '#B9D6BC' : '#D8C599').textAlign(TextAlign.Center)
          .width('100%').padding({ top: 4 })
      }.width('76%').padding({ top: 28, bottom: 44 }).constraintSize({ minHeight: '100%' })
    }.width('100%').height('100%').backgroundColor('#0C1917').scrollBar(BarState.Off)
  }
}

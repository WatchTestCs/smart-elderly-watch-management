# 岁宁守护 WATCH 5 手表端

本工程是独立的 HarmonyOS 原生穿戴应用，使用 ArkTS、ArkUI 和 Stage 模型，目标为 HUAWEI WATCH 5。当前交付用于 DevEco 穿戴模拟器联调：健康数值全部为明确标注的模拟数据，不读取真实传感器，也不实现跌倒识别或医疗诊断。

## 本版功能

- 深色圆屏界面：首页、记录、设置；滚动访问各项操作。
- 后台设备认证、设备绑定码绑定长者、心跳。
- 手动生成模拟心率和步数，通过 `healthBatch` 上传，不伪造血压或睡眠。
- 本机保存最多 200 条待上传模拟采样；稳定记录 ID 支持服务端去重。上传前校验服务器、设备、长者和绑定版本，绑定变化时保留并拒绝补传旧记录。
- SOS 二次确认；不确定结果保留同一 requestId，重试同一次求助不重复建单。页面区分后台接收与通知送达。
- 仅地址、设备号、模拟记录与未确认求助保存在应用私有 Preferences。设备密钥只在内存中使用，重启需重新输入；不包含管理员密码。

## 环境

本机已使用 DevEco Studio 6.1.1.280、自带 SDK 6.1.1 API 24 和 Node 18 编译。兼容版本配置为 HarmonyOS 5.1 API 18；已在 API 24 Wearable 模拟器安装并通过命令行启动。WATCH 5 真机安装、个人签名、实际系统版本兼容性和传感器权限尚未验证。

本工程已随网页端、Go 后端和桌面壳一起放在同一仓库分支，目录为仓库根的 `SuiningWatch/`。
原始独立目录 `E:\DevEco Test\SuiningWatch` 保留。DevEco Studio 选择 Open，打开克隆仓库中的
`SuiningWatch` 文件夹，不是只打开 entry，也不把它当作 Vue 工程。

## 构建和启动

在 PowerShell 7 中：

```powershell
cd '<仓库目录>\SuiningWatch'
./scripts/build.ps1
```

脚本复用 DevEco 自带构建工具，在本项目 node_modules 中创建依赖目录链接，不改动 IDE 安装目录。其他安装位置通过 `-DevEcoHome` 指定。源码中的 hvigorfile 保留标准包名，便于 IDE 打开。

先在 DevEco 的 Device Manager 启动 Wearable 模拟器，再运行：

```powershell
./scripts/run-emulator.ps1 -Target '127.0.0.1:5555' -BackendPort 8796
```

脚本通过 HDC 把模拟器 `127.0.0.1:18796` 转发到电脑的 `127.0.0.1:8796`，不需要将本机数据库或 API 公开到局域网。此端口对应当前新版 Go 服务；旧版 Java 的 8795 不能替代新版健康采样接口。

HAP 输出：`entry/build/default/outputs/default/entry-default-unsigned.hap`。这是模拟器调试包，真机需通过个人开发者账号签名，不要将未签名包作为真机发布包。

## 与管理平台联调

1. 确认新版 Go 服务已运行，`http://127.0.0.1:8796/api/v1/service/status` 中 databaseReady 为 true。
2. 在管理平台先建立一个明确命名的测试长者，再在设备管理登记模拟手表。保管创建时仅返回一次的设备密钥。
3. 在手表“设置”填写地址 `http://127.0.0.1:18796`、后台分配的设备号和设备密钥。密钥不是管理员登录密码。
4. 点击“保存地址并认证”。未绑定时，填写后台生成的**设备绑定码**，不要使用家属注册邀请码。
5. 首页“生成模拟采样”，记录页“上传记录”，在后台健康采样页面核对。接口采用新健康采样链路，旧版兼容的心率血压列表未必展示相同记录。
6. 点击“SOS 模拟求助”并确认，在运营总览中检查新建照护任务。本操作会写入真实后端测试数据；如果后台配置了真实通知渠道，可能触发该渠道。请仅使用专用测试长者和测试通知配置。

## 测试

```powershell
npm ci
npm test
```

Node 协议测试使用 package-lock.json 固定的 TypeScript 5.9.3，可独立于 DevEco 运行，
CI 将其作为必需检查并上传 watch-<SHA> 证据。它只转译生产 Protocol.ets 并执行协议断言，
不替代使用 DevEco SDK 的完整 ArkTS/ArkUI 原生构建。SDK 工具链仍由 build.ps1 管理。

测试直接加载生产使用的 Protocol.ets，检查绑定隔离、重试 ID、采样语义及队列容量，不替代模拟器交互与真实传感器验证。

`tests/backend.e2e.cjs` 仅面向全新隔离测试后端。需要通过环境变量提供 `WATCH_TEST_URL`（仅允许本机 HTTP 地址）和 `WATCH_TEST_SETUP_TOKEN`；脚本验证空库后创建临时管理员、模拟监护与长者、设备和模拟 SOS。不要对已有业务库运行。绑定码创建明确传入 `expireDays: '1'`。

原独立工程 2026-10-25 的记录：6 项协议单元测试、6 项隔离 Go + MySQL 协议检查通过，
覆盖认证、绑定、心跳、采样入库/去重、SOS 去重及重绑定拒绝旧批次。原 artifacts、缓存、
安装包和私有配置未导入 Git；不能把这些历史结果当作当前分支已经通过的证明。
本次整合计划与新验收入口见 [同分支整合记录](../docs/WATCH_INTEGRATION.md)。
E2E 由电脑端使用相同 Protocol.ets 构造请求，不是模拟器完整点击流程的证明。

本次创建的 `suining_watch_protocol_20261025`、`suining_watch_protocol_20261025b` 为独立测试库，不连接网页当前使用的 `suining_preview`。测试服务使用 8797，测试完成后停止；测试库保留供核查。日常手表仍经 18796 转发连接新版服务 8796。

## 当前限制

- 无后台持续采集、无自动上传；手动采样和手动补传便于清晰控制测试写入。
- 无真实健康传感器、GPS、睡眠和血压接入；模拟器数据不证明硬件能力。
- 关闭应用后尚未重新认证前只能本地预览，不能产生有归属的离线记录。最近一次已认证绑定的离线记录在补传时重新校验。
- 健康上传由服务端原子校验 bindingVersion；SOS 接口目前没有同等的服务器端绑定版本条件，因此重绑定时应停止求助操作，待重新认证后再发起。
- 本地存储仅用于模拟数据。接入真实敏感采样前应补充加密存储、密钥安全保管和退出清理策略。
- 目前仅完成命令行编译、安装、启动与协议验证；未完成手表界面视觉验收。若圆屏文字或布局需要调整，应以真实模拟器交互结果为依据。

## 工程结构

```text
AppScope/                              应用标识和图标
entry/src/main/ets/pages/Index.ets      圆屏原生界面
entry/src/main/ets/model/Protocol.ets   采样与绑定隔离规则
entry/src/main/ets/services/WatchApi.ets   HTTP 客户端
entry/src/main/ets/services/WatchStore.ets 本机模拟记录
scripts/                               构建与模拟器启动脚本
tests/                                 协议测试
```

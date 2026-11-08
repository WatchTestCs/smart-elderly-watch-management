# 测试分层 · 当前 preview

## Go 后端

在 `智慧养老手表管理系统/backend` 目录执行：

```sh
go test -count=1 ./...
go vet ./...
```

需要数据库集成测试时，配置 `TEST_MYSQL_DSN` 与 `TEST_DATABASE_URL`，使用专用、可销毁的测试实例；MySQL 测试账号需有创建和删除随机测试库的权限。竞态测试：

```sh
go test -race -count=1 -timeout 600s ./...
```

未配置测试数据库时，相关集成用例可能跳过；不得将跳过报告为真实数据库验收通过。禁止对生产数据库运行自动测试。

## Vue 前端

在 `智慧养老手表管理系统/frontend` 目录执行：

```sh
npm ci
npm test
npm run typecheck
npm run api:check
npm run test:api-drift
npm run build
```

浏览器端 Playwright 测试：`npm --prefix qa ci` 后执行 `npm --prefix qa test`。模拟接口测试不能替代真实后端集成测试。

## HarmonyOS 手表端

在 `SuiningWatch` 目录执行：

```sh
npm ci
npm test
```

协议测试基于适配器转译生产 ArkTS 模型，不等价于 DevEco 原生编译、模拟器交互、传感器权限验证或真机验收。原生构建要求与当前验证边界见 [手表工程说明](../SuiningWatch/README.md)。

## Windows 桌面端

按 [Windows 构建说明](BUILD.md)准备 MSVC、Windows SDK、Rust、Go、Node.js 与 PowerShell。执行 Tauri 构建、sidecar 生命周期、安装/卸载及 WebView2 缺失补装检查。编译成功不能替代真实安装环境和可视交互验收。

## 生产安全边界

- 不对生产数据库执行自动化测试或迁移演练。
- 业务写入失败不得自动重放；分析 outbox 按幂等语义重试。
- 手表协议测试不代表真实硬件、SIM/网络或通知网关验收。
- 网关接受请求不等于通知送达；送达和现场处置必须分别验证。
- 当前提交的 CI 状态以 GitHub Actions 对应提交的完整检查结果为准。

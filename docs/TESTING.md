# 测试分层 · 当前 preview

## Go 后端

在 `backend` 目录执行：

```sh
go test -count=1 ./...
go vet ./...
```

需要数据库集成测试时，配置 `TEST_MYSQL_DSN` 指向**独立、可销毁**的 MySQL 8.4 测试实例；测试账号需有创建和删除随机测试库的权限。竞态测试：

```sh
go test -race -count=1 -timeout 600s ./...
```

未配置测试数据库时，相关集成用例可能跳过；不得将跳过报告为真实数据库验收通过。禁止对生产数据库运行自动测试。

### 测试组织与快照实验

普通测试默认创建独立随机 schema，执行完整迁移并清理。只有需要测量普通 handler fixture 的准备成本时才设置 `TEST_MYSQL_FIXTURE_SNAPSHOT=1`；该模式复用进程内的 pristine SQL 快照，每个测试仍恢复到新建空 schema，并验证迁移账本和结构指纹。显式迁移与协调恢复测试继续走完整迁移。需要与实例匹配的 MySQL 客户端或独立测试容器。

在已配置隔离数据库的 PowerShell 7 环境中比较两种模式：

```powershell
$env:TEST_MYSQL_FIXTURE_SNAPSHOT = '1'
go test -race -count=1 -v -run '^TestMySQLSnapshotIsolationAndCost$' ./internal/db/dbtest
go test -race -count=1 ./internal/handlers
Remove-Item Env:TEST_MYSQL_FIXTURE_SNAPSHOT
go test -race -count=1 ./internal/handlers
```

仅在行为一致且真实耗时对照证明有收益后考虑默认启用。不要把静态减少初始化次数当作实际 CI 加速。相同输入与断言已有真实行为覆盖时，不再重复检查 SQL 拼写；保留独立授权、事务、竞态、恢复、加密与边界断言。复用同一测试 fixture 时必须保留状态隔离及所有关键断言。


## Vue 前端

在 `frontend` 目录执行：

```sh
npm ci
npm test
npm run typecheck
npm run lint
npm run api:check
npm run test:api-drift
npm run build
```

在仓库**根目录**运行浏览器端 Playwright 验收：`npm --prefix frontend/qa ci`，随后执行 `npm --prefix frontend/qa test`。在 `frontend/` 内运行时则使用 `npm --prefix qa ...`。根目录 `tests/ui/` 是独立的 UI 质量检查，具体启动方式以 `.github/workflows/ui-quality.yml` 为准。模拟接口测试与截图不能替代真实后端、WebView2 及硬件验收。

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
- 未确认成功的用户写入不得自动重放；通知 outbox 或其他明确设计为幂等的后台任务可按各自协议重试。
- 手表协议测试不代表真实硬件、SIM/网络或通知网关验收。
- 网关接受请求不等于通知送达；送达和现场处置必须分别验证。
- 当前提交的 CI 状态以 GitHub Actions 对应提交的完整检查结果为准。

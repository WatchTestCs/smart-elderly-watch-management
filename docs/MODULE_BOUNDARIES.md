# preview 模块依赖与公开接口

基线：`60ed3bdf7e44b389c99f23691027b9620073b150`（2026-11-17 审查）。这是针对现有模块化单体的渐进约束，不要求改动 API、数据库模式或拆分服务。

## 实测依赖图与风险

- Go：`go list` 可解析的 60 个内部包，无编译级导入环；`handlers` 直接依赖 48 个内部包。主依赖方向是入口 → workflow/repository/policy → DB/存储。
- Vue/TS：审查时 `auth.ts` ↔ `domain/permissions.ts` 含类型导入构成双向依赖（非运行时环）；`domain/setup.ts` 反向导入 `api/httpContract.ts`；`composables/useRecordCharts.ts` 动态导入根目录图表实现。
- Go：`vitalpolicy` 曾依赖 `vitalrepo.Alert`。本次将纯数据契约移至 `vitalpolicy.Alert`，`vitalrepo.Alert` 保留类型别名兼容。
- HarmonyOS：21 个 ETS 文件、32 条已解析相对导入，无静态导入环。Tauri 当前入口集中在 `desktop/src/main.rs`，本次未修改 Rust 模块布局。

| 风险 | 涉及模块 | 影响及决策 |
| --- | --- | --- |
| P1 中（已修） | 前端 `auth` / `domain/permissions` | 类型双向依赖；共享 `SessionAccount` 下沉为领域类型并保持旧类型导出 |
| P1 中（已修） | `domain/setup` / `api/httpContract` | 领域层依赖网络错误实现；分类器移至 API 适配层，保留一次性写入失败文案和用例 |
| P1 中（已修） | `vitalpolicy` / `vitalrepo` | 纯告警渲染依赖持久化模型；调整依赖方向，仓库对外类型别名兼容 |
| P1 中（保留） | `authrepo.InitializeAdmin` → `carerepo.SetDuty` | 第一次创建管理员、授予主管岗位与审计必须在同一事务中；禁止在无并发回归时拆分 |
| P1 中（保留） | `devicerepo.Create/CompleteBinding` → `telemetryrepo.EnsureDefaultWatchProfile` | 初次绑定和阈值初始化要求事务原子性；禁止提早提交或重复写入 |
| P2 中（后续） | `handlers` → 48 个内部包 | 较宽的编排面易混合权限、持久化、协议；仅逐条迁移已经验证的事务用例到相应 workflow |
| P2 低（后续） | `config` → `blob.S3Options` 等 | 配置对象与实现类型耦合；没有兼容性收益时暂不新增配置 DTO |
| P2 低（后续） | Tauri 单入口与设备 UI 类型 | 单入口和 `composables/deviceModels` 混合 UI 状态/契约；只在新用例证明有问题时拆分 |

## 目标依赖方向

- Go：`cmd/server` → `handlers` → `*workflow` → `*repo`；workflow 可以依赖 `*policy`，repo 可以依赖数据库基础设施与纯领域契约。policy 不依赖 repo/handler/workflow。请求权限仍须在业务事务内核验。
- Vue：`App/router/views` → `components/composables` → `api`；各层可以依赖 `domain` 和纯 `utils`。`domain` 禁止导入网络、UI、状态管理与组件；`charts` 不反向依赖 composables。前端类型依赖与动态导入都应无环。
- Tauri：只管理可信原生窗口、进程、权限、IPC 和退出；不吸收后端 SQL 与业务规则。
- HarmonyOS：`pages` 调度 `services`，`model` 提供协议数据类型；手表端不实现服务器授权策略。

## 模块公开接口（以源码导出为准）

| 模块 | 保持或提供的入口 | 非公开实现不应被跨层依赖 |
| --- | --- | --- |
| `frontend/src/domain/session.ts` | `SessionAccount` | `auth.ts` 的拦截器、会话缓存和路由 |
| `frontend/src/domain/permissions.ts` | `isRestrictedAdmin`、`mayReadCareAudit`、`mayManageCareStaff` | 页面、Axios、登录态存储 |
| `frontend/src/api/httpContract.ts` | `apiErrorMessage`、`transportOutcome`、`readErrorMessage` | HTTP 内部解码过程 |
| `frontend/src/api/setupFailure.ts` | `setupFailureMessage` 与原 `SETUP_*` 常量 | 管理员创建流程仅由调用方编排，失败不能重放写入 |
| `frontend/src/composables/useDevices.ts` | `useDevices`、原有 `DeviceRecord` 类型再导出 | 刷新计时器、请求取消和视图内部状态 |
| `backend/internal/vitalpolicy` | `Alert`、`DescribeRatepreAlert`、`Thresholds`、`Reason` | SQL/HTTP |
| `backend/internal/vitalrepo` | `Alert`（兼容别名）、`FindAlert`、`FindLatestPending` | SQL 查询文本 |
| `backend/internal/deviceworkflow` | `Bind`、`SOS`、`Authenticate` 等业务用例 | repo 的 SQL、锁与内部字段 |
| `backend/internal/careworkflow` | `BeginAuthorized`、`AssignDuty`、`RecoverSupervisor` 等事务用例 | 独立 Handler 的事务外授权缓存 |

保留 `auth.ts` 的 `SessionAccount` 类型再导出，和 `vitalrepo.Alert` 的 Go 类型别名，已有使用者不必立即迁移。

## 自动边界门禁

- 前端：`npm run check:boundaries` 用既有 TypeScript AST 扫描 TS、Vue 的静态、动态和类型导入，禁止领域/适配/组件反向依赖，并拒绝导入环；已加入 `typecheck` 和 `typecheck:stable`，由现有前端 CI 执行。
- Go：`python3 scripts/check-go-module-boundaries.py` 调用标准 `go list`，禁止 repo → workflow/handlers、policy → repo/workflow/handlers 及新增跨业务 repo 依赖；现有两条具备事务语义的跨 repo 边显式记录为例外。已加入现有后端 CI。
- Go 导入环由 `go list` / 编译器阻断。别名重导出不改变旧函数签名；数据库约束和 SQL 语义保持不变。

## 下一步最小迁移

1. 在独立 MySQL 上验证首次主管初始化、并发岗位变更及撤权事务，随后考虑将 `InitializeAdmin` 的跨业务编排迁至 workflow。
2. 用真实绑定码消耗、阈值幂等、重绑隔离及失败回滚证明可以把 `EnsureDefaultWatchProfile` 编排移至 `deviceworkflow`。
3. 每次只处理一个 Handler 的职责与权限边界，不批量替换 SQL 或增加泛型 Repository。
4. 持续以同一准确 HEAD 运行 Go 后端、前端严格类型检查、171 项前端回归及构建、CodeQL、必需 CI；涉及设备原生行为再执行真机验收。

尚未完成上述两项数据库事务搬迁，因此不能宣称后端已彻底按理想分层隔离。

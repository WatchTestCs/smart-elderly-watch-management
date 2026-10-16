# 项目交接指南

本文件是新维护者进入 `preview` 分支的第一入口。目标是让接手者在不依赖口头说明的情况下，能够判断项目边界、启动开发环境、运行质量门禁、修改 UI、处理数据、打包发布，并知道哪些结论已经有证据、哪些仍未验证。

## 1. 先记住这些规则

- 当前工程化目标分支是 **`preview`**；不要把当前交接改造直接同步到 `main`。
- 不要通过“大重写”换取代码看起来整洁。先用测试锁定行为，再渐进拆分。
- 不要把历史验收当成当前 commit 自动通过。证据边界见 [TESTING.md](TESTING.md)。
- 不要削弱 Electron 安全边界来解决开发问题。架构不变量见 [ARCHITECTURE.md](ARCHITECTURE.md)。
- 不要默认删除 `%APPDATA%\岁宁`。它是用户数据，不是安装缓存。
- 不要在普通 PR/CI 中读取签名私钥。
- 健康数据“没有新记录”不等于“正常”；UI 不应做未经依据的医疗诊断。

## 2. 10 分钟导航

| 你要做什么 | 先看 |
| --- | --- |
| 理解产品和用户 | 根目录 `README.md` |
| 第一次启动/使用 | [SETUP.md](SETUP.md) |
| 理解系统边界 | [ARCHITECTURE.md](ARCHITECTURE.md) |
| 理解为什么这样设计 | [decisions/](decisions/README.md) |
| 修改 UI/UX | [UI_UX.md](UI_UX.md) |
| 运行测试 | [TESTING.md](TESTING.md) |
| 自行构建 | [BUILD.md](BUILD.md) |
| 看历史验收 | [VALIDATION.md](VALIDATION.md) |
| 看发布边界 | [PUBLICATION.md](PUBLICATION.md) |
| 查文件职责 | [FILES.md](FILES.md) |

## 3. 代码地图

```text
智慧养老手表管理系统/
├── oldIot/      Vue 3 / Element Plus / Vite UI
├── careold/     Spring Boot / Security / JDBC / H2
├── desktop/     Electron 生命周期、运行时、安装与签名
└── watch-sim/   模拟设备联调

.github/workflows/
├── ci.yml
├── reusable-frontend.yml
├── reusable-backend.yml
└── ui-quality.yml

tests/ui/
└── ui-smoke.mjs
```

### 前端

优先关注：

- `src/router.js`：路由和角色边界；
- `src/auth.js`：会话状态；
- `src/views/`：业务页面；
- `src/components/`：共享组件；
- `src/assets/css/commercial.css`：当前集中视觉样式，后续逐步拆分。

当前较大的 SFC（如设备、健康、人员相关页面）属于维护热点，但不要直接重写。

### 后端

优先关注：

- `security/SecurityConfig.java`：访问控制、CSRF、CSP；
- controller/service/dao：业务与事务边界；
- `application.properties`：loopback、H2、session 与连接池；
- `src/test/`：已有安全、事务和数据访问测试。

### 桌面

优先关注：

- `desktop/main.cjs`：最高权限运行时边界；
- `desktop/runtime-lock.json`：固定运行环境；
- `desktop/integrity.json` 或构建生成的完整性清单；
- `desktop/build.ps1` 与安装脚本：后续流水线拆分重点。

## 4. 第一天应该运行什么

### 核心 CI 对应本地命令

前端：

```powershell
cd 智慧养老手表管理系统/oldIot
npm ci
npm run build
npm audit --omit=dev --audit-level=high
```

后端：

```powershell
mvn -Dnet.bytebuddy.experimental=true -f 智慧养老手表管理系统/careold/pom.xml clean verify
```

GitHub 上的 `Preview CI / required` 必须通过。

### UI 基线

`.github/workflows/ui-quality.yml` 当前会验证：

- Chrome 真实渲染；
- 1440×1000 桌面登录页；
- 390×844 移动端登录页；
- 390×844 移动端大字模式；
- Accessibility Tree 可访问名称；
- 可见交互目标 ≥44px；
- 桌面登录页粗粒度视觉回归；
- 截图和浏览器日志 artifact。

这只是 UI 自动化的起点，不代表登录后的全流程已经自动覆盖。

## 5. 修改 UI 的工作流

1. 先确定用户任务，而不是先改颜色/圆角。
2. 检查标准状态：loading / empty / error / normal / long-content / destructive-confirmation。
3. 检查桌面、窄屏和大字模式。
4. 优先复用现有组件，避免复制新的一套页面样式。
5. 需要新视觉常量时，优先进入 design token，而不是继续散落 magic value。
6. 运行生产构建和 UI Quality。
7. 视觉基线只有在“预期变化”且人工确认后才能更新；不能通过自动更新截图让 CI 变绿。

详细契约见 [UI_UX.md](UI_UX.md)。

## 6. 修改安全/架构的工作流

以下改动不是普通重构：

- 改 Electron sandbox / Node 权限；
- 改后端 bind address；
- 改桌面 token 传递方式；
- 改数据库位置或 schema 生命周期；
- 改家属/管理员授权模型；
- 改安装/升级/卸载的数据保留语义；
- 改发布分支模型。

这类改动必须：

1. 先读 [ARCHITECTURE.md](ARCHITECTURE.md)；
2. 检查 [decisions/](decisions/README.md)；
3. 新增或 supersede ADR；
4. 增加能锁定新不变量的测试；
5. 再修改实现。

## 7. 数据与恢复

默认用户数据目录：`%APPDATA%\岁宁`。

目前最稳妥的备份方式仍是：

1. 完全退出软件；
2. 确认 Java 后端退出；
3. 复制完整数据目录；
4. 在隔离环境恢复验证。

不要只复制 H2 主文件而忽略图片和其他相关数据。未来独立 runbook 会把备份/恢复和 schema migration 固化。

## 8. 发布交接原则

当前 preview 发布必须保持：

- 唯一 commit 可追溯；
- 核心 CI 通过；
- UI Quality 通过；
- Windows 桌面 smoke 通过；
- 安装/覆盖升级/卸载边界验证；
- 签名与 checksum 可验证；
- 不把自签名描述为受公众信任；
- 明确列出尚未验证的能力。

普通 CI 生成的未签名包和最终签名发布资产必须区分。

## 9. 当前已知缺口

接手者不要重新“发现”以下已知问题：

- Windows 10 尚未实机验收；
- 真实手表端到端联调未完成；
- 真实短信发送未完成；
- 完整性能基准未建立；
- 完整第三方渗透测试未完成；
- UI 自动化目前重点覆盖登录页，业务主流程需要继续扩展；
- 大型 Vue SFC 和 `commercial.css` 仍需要渐进拆分；
- 桌面构建已拆分为 build/package/release 阶段；完整 Windows 打包与安装生命周期仍需在 release candidate 阶段持续验证。

## 10. Definition of Done

### 普通代码 PR

- 生产构建通过；
- 后端受影响测试通过；
- 没有新增未解释的 warning/error；
- 文档与行为一致。

### UI PR

额外要求：

- UI Quality 通过；
- 键盘可操作；
- 大字和窄屏不丢任务能力；
- loading/empty/error 状态合理；
- 预期视觉变化经过人工确认。

### 架构 PR

额外要求：

- ARCHITECTURE 或 ADR 更新；
- 新不变量有自动测试；
- 数据迁移/兼容性影响写清。

### Release Candidate

额外要求：

- Windows 桌面真实启动；
- 空数据首次设置；
- 登录和核心流程；
- 重启数据保留；
- 安装/升级/卸载；
- 发布资产摘要与签名；
- 明确已验证/未验证边界。

## 11. 出问题时先看哪里

| 现象 | 第一排查点 |
| --- | --- |
| UI 构建失败 | `oldIot/package-lock.json`、Node 版本、Vite 输出 |
| UI Quality 失败 | Actions artifact、Chrome log、截图差异 |
| 登录/权限异常 | `SecurityConfig.java`、session/CSRF、角色路由 |
| 软件打不开 | `startup-error.log`、`server.log`、integrity/runtime lock |
| 数据不见 | 先确认实际 `userData`，不要初始化/删除数据库 |
| 覆盖升级异常 | 安装脚本与数据目录边界 |
| 打包结果不可追溯 | commit SHA、版本源、构建输出目录、checksum |

交接目标不是“任何人立刻理解所有代码”，而是让下一位维护者不会因为缺少隐含知识而破坏用户数据、安全边界或 UI 可用性。

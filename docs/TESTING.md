# 测试与质量门禁

本文件定义 `preview` 分支可以重复执行的测试、历史验收证据与后续新增测试的最低要求。目标是让维护者能够从仓库本身判断“什么已经自动证明、什么仍需人工验证”，避免把一次性的验收记录误当成持续质量保证。

## 一条原则

任何文档中写成“通过”的检查，都必须满足以下两者之一：

1. 仓库中存在可重复执行的命令或 CI workflow；
2. 明确标记为人工验收/历史验收，并记录环境、日期和边界。

不得把没有重新执行的历史结果描述成当前提交已经再次通过。

## 当前自动化基线

`preview` 的核心 CI 入口是 `.github/workflows/ci.yml`。此外还有：

- `.github/workflows/ui-quality.yml`：真实 Chrome UI/a11y/视觉基线；
- `.github/workflows/desktop-contract.yml`：Windows 构建脚本和版本契约；
- `.github/workflows/security.yml`：CodeQL 与 PR dependency review；
- `.github/workflows/preview-candidate.yml`：手动 unsigned Windows candidate + SBOM/provenance。

### 前端

本地执行：

```powershell
cd 智慧养老手表管理系统/oldIot
npm ci
npm run build
npm audit --omit=dev --audit-level=high
```

CI 使用 Node.js 22，并严格按 `package-lock.json` 安装。当前自动化只证明依赖可安装、生产构建成功，以及生产依赖没有达到 high/critical 门槛的已知 npm advisory。

它**尚不等价于**完整 UI 功能验收。浏览器级 UI 基线由独立的 `UI Quality` workflow 执行。

### 后端

本地执行：

```powershell
mvn -Dnet.bytebuddy.experimental=true -f 智慧养老手表管理系统/careold/pom.xml clean verify
```

CI 使用 Temurin Java 21，并执行 Maven 完整 `verify` 生命周期。现有测试覆盖数据库、事务、匿名访问、家庭账号密码、安全集成、图片接口、设备告警查询与公开接口限流等已有测试类。

## 历史验收与当前 CI 的关系

`docs/VALIDATION.md` 记录 2026-10-15 的预览版验收，包括：

- 后端 38 项测试；
- 296 项常规界面检查与 10 项首次设置检查；
- 安装后的真实程序 53 项检查；
- 安装、覆盖升级、单实例、退出和卸载生命周期；
- 签名与发布文件摘要。

这些结果在未被自动脚本重新执行前属于**历史验收证据**。

当前已经自动化的 UI 基线位于 `.github/workflows/ui-quality.yml` 与 `tests/ui/ui-smoke.mjs`，会在真实 Chrome 中验证：

- 1440×1000 登录页桌面基线；
- 390×844 登录页移动端；
- 390×844 移动端大字模式；
- Accessibility Tree 中关键交互控件/标题具有可访问名称；
- 可见交互目标不小于 44px；
- 桌面登录页与已提交基线截图的粗粒度 RMSE 视觉差异门槛；
- 每次运行上传截图和浏览器日志作为短期证据。

这仍不是完整 E2E。登录后的管理员/家属主流程和 Electron 安装态 UI 后续继续纳入自动化。

## 测试层级

| 层级 | 用途 | 是否阻断 PR |
| --- | --- | --- |
| Static/build | 锁文件安装、编译、生产构建 | 是 |
| Static security | CodeQL Java + JavaScript/TypeScript | 是 |
| Backend unit/integration | 权限、数据库、事务、接口语义 | 是 |
| Frontend unit/component | UI 状态和可复用逻辑 | UI 改动时是 |
| Accessibility | 键盘、语义、axe 自动检查 | UI 改动时是 |
| Visual regression | 布局、字号、颜色、响应式基线 | UI 改动时是 |
| Browser E2E | 登录、建档、设备/家属核心路径 | 核心流程改动时是 |
| Electron smoke | 真实桌面壳、随包 Java、重启持久化 | 发布候选是 |
| Installer lifecycle | 安装、覆盖升级、卸载保留数据 | 发布是 |

## 测试数据规则

自动测试数据必须可重复：

- 使用固定账号、固定日期、固定设备号与固定记录；
- 不依赖真实短信、真实手表或公网业务服务；
- 测试数据库与用户真实 `%APPDATA%\岁宁` 完全隔离；
- 不把测试数据库、账号凭据、私钥或本机绝对路径提交到仓库；
- 截图测试固定字体、viewport、语言和动画状态。

## 失败处理

CI 失败时不要通过重跑掩盖真实问题。先判断失败属于：

1. 确定性代码/测试失败；
2. 外部依赖下载失败；
3. 已知且有证据的 flaky test。

只有第 2 或第 3 类允许重跑；重复出现的 flaky test 必须建档并修复，不能长期依赖 retry。

## 发布前最小门槛

`preview` 发布至少要求：

- 核心 CI 全绿；
- Security/CodeQL 全绿；
- UI 变更通过 UI quality workflow；
- 后端安全与数据库测试通过；
- Windows 桌面 smoke test 通过；
- 安装/升级/卸载边界按发布 runbook 验证；
- 发布资产生成 SHA-256，并能追溯到唯一 `preview` commit。

详细发布门槛将在 `docs/RELEASE.md` 和 `docs/runbooks/` 中定义。

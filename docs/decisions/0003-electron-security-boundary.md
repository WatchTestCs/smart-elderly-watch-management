# ADR-0003：Electron Renderer/Main 安全边界

- Status: Accepted
- Date: 2026-10-16

## Context

Renderer 展示和处理用户可见数据，但不应拥有本机任意执行能力。首次设置和桌面关闭接口又需要证明请求来自受控桌面进程。

## Decision

BrowserWindow 保持以下边界：

- `nodeIntegration: false`
- `contextIsolation: true`
- `sandbox: true`
- `webSecurity: true`
- 默认拒绝权限申请；
- 默认拒绝新窗口、WebView、下载和跨 origin 导航。

桌面主进程启动时生成随机 `X-Suining-Desktop` 令牌，仅通过 Electron 网络层向本机 Spring origin 注入。令牌不暴露给页面 JavaScript、localStorage 或普通业务状态。

随包 Java、服务 JAR 等关键组件在启动前按 `integrity.json` 验证摘要；运行时版本由 `runtime-lock.json` 固定。

## Consequences

Renderer 被视为低权限 UI 层；即使出现前端脚本问题，也不应直接获得 Node/文件系统能力。

任何要求开启 NodeIntegration、关闭 sandbox、允许任意导航、把桌面 token 暴露到 JS 的变更，都属于架构变更，必须新增 ADR 和安全测试。

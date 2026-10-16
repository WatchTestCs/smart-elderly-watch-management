# ADR-0001：Electron + 本机 Spring 服务

- Status: Accepted
- Date: 2026-10-16

## Context

项目同时需要桌面安装体验、Vue 界面、既有 Java/Spring 业务代码、H2 本机持久化和明确的安全边界。把全部后端逻辑迁入 Electron 会扩大重写范围，并让数据库、权限和事务逻辑与桌面主进程耦合。

## Decision

桌面版采用 Electron 主进程管理生命周期，随包启动 Java/Spring Boot 服务。Spring 服务默认绑定回环地址和动态端口；Electron 等待就绪文件后，仅加载该本机 origin。Vue Renderer 通过同源 HTTP 调用业务接口。

Electron 负责：

- 随包组件完整性校验；
- Java 进程启动、就绪和关闭；
- BrowserWindow 安全策略；
- 桌面专用随机令牌注入；
- 单实例与桌面生命周期。

Spring 负责：

- 身份与权限；
- 业务接口；
- 数据库事务；
- 数据持久化；
- 设备、家属与照护领域规则。

## Consequences

优点：

- 保留 Spring Security、JDBC 与既有测试；
- Renderer 不需要 Node 权限；
- 后端可独立执行集成测试；
- 生命周期责任清晰。

代价：

- 安装包需要携带 Java Runtime；
- 必须处理 Java 子进程、动态端口、就绪与异常退出；
- Electron 与 Spring 间存在本机 HTTP 边界，需要专门防止非桌面进程调用桌面专用接口。

## Rejected alternatives

- 将全部后端迁入 Electron/Node：迁移风险过高，削弱现有 Java 测试与事务模型。
- 固定公开端口：增加端口冲突与不必要暴露。
- 远程云服务作为默认后端：改变当前“本机优先”的隐私和部署模型。

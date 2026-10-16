# Architecture Decision Records

本目录记录对项目长期维护有约束力、且不适合只写在代码注释里的架构决定。

## 规则

- 每个难以逆转或会影响多个模块的决定使用一个 ADR。
- ADR 一旦标记为 `Accepted`，不通过改写历史来改变结论；若决定变化，新建 ADR 并在旧 ADR 中标记 `Superseded by ADR-xxxx`。
- ADR 记录“为什么这样做”，`ARCHITECTURE.md` 记录“系统现在怎样工作”。
- PR 若改变本文列出的架构不变量，必须同时更新架构文档或新增 ADR。

## 索引

- [ADR-0001：Electron + 本机 Spring 服务](0001-desktop-local-service-architecture.md)
- [ADR-0002：H2 文件数据库与用户数据目录](0002-local-h2-data-storage.md)
- [ADR-0003：Electron Renderer/Main 安全边界](0003-electron-security-boundary.md)
- [ADR-0004：面向老年照护场景的 UI 设计契约](0004-ui-design-system.md)
- [ADR-0005：preview 预发布与交付模型](0005-preview-release-model.md)

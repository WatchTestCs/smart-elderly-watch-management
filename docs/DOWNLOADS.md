# 2.1.0 下载与历史公开附件 / Downloads and history

核对日期：**2026-10-22**。此索引根据 GitHub Release 元数据核对版本、文件名、大小和下载地址；未重新安装或执行历史二进制，也不据此宣称兼容当前数据库。

## 当前版本 2.1.0 / Current version

[v2.1.0 Release](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) 的安装版、免安装版和源码来自同一 `preview` 提交，精确来源及检查见随包清单。

| 文件 | 用途 |
| --- | --- |
| [Suining-Setup-2.1.0-win-x64.exe](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Setup-2.1.0-win-x64.exe) | Windows x64 安装；WebView2 缺失时联网补装 |
| [Suining-Portable-2.1.0-win-x64.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Portable-2.1.0-win-x64.zip) | 完整解压并运行 suining.exe，保留 suining-server.exe；需要 WebView2 |
| [Suining-Source-2.1.0.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/Suining-Source-2.1.0.zip) | 完整源码，包含字体及构建脚本 |
| [SHA256SUMS.txt](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/SHA256SUMS.txt) · [release-manifest.json](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.1.0/release-manifest.json) | 下载摘要、源码提交及实际打包检查 |

Current 2.1.0 includes unsigned Windows Setup, portable and full source packages. Databases are independently deployed. See [upgrade notes](releases/2.1.0.md); historical executables retain their original architecture and limitations.

## 历史版本索引 / Release history

| Release | 主要内容 / Scope | 状态 / Status |
| --- | --- | --- |
| [v2.1.0](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.1.0) | 双库 Windows 安装、免安装及完整源码；见随包来源清单 | 预览 / prerelease |
| [v2.0.6](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.6) | 历史 2.0.6 双库整合源码 ZIP + SHA-256，无安装包；不含标签发布后的 `preview` 更新 | 源码预览 / prerelease |
| [v2.0.5-optimized.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.5-optimized.1) | 架构优化历史源码；含字体版与无字体版；旧 PG 单库 | 预览 / prerelease |
| [v2.0.5-cleanup.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.5-cleanup.1) | 死代码清理修订源码；旧 PG 单库 | 预览 / prerelease |
| [v2.0.5](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.5) | 值守可靠性与权限治理源码；旧 PG 单库 | 预览 / prerelease |
| [v2.0.4](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.4) | 求助照护闭环源码预览；旧 PG 单库 | 预览 / prerelease |
| [v2.0.3](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.3) | 源码结构拆分历史版本；旧 PG 单库 | 预览 / prerelease |
| [v2.0.2-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.2-preview.1) | Windows 安装／免安装、源码及 Linux 独立附件 | 预览 / prerelease |
| [v2.0.1-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.1-preview.1) | 早期 Windows 预览二进制，使用当时的运行时与配置 | 预览 / prerelease |
| [v2.0.0-preview.1](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.0-preview.1) | 早期桌面预览及源码；不套用当前构建与配置 | 预览 / prerelease |
| [archive-2026-10-13](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/archive-2026-10-13) | 原始 RAR 源码归档，供历史复核，不作当前启动入口 | 归档 / archive |

## 历史源码附件：2.0.6 双库整合

2026-10-19 发布的 [v2.0.6 源码预览](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/tag/v2.0.6) 附带以下三项：

| 附件 | 适用范围 |
| --- | --- |
| [Suining-Source-2.0.6-preview.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.6/Suining-Source-2.0.6-preview.zip) | 标签快照源码，约 **27.27 MiB**；不包括 Windows 可执行安装程序，也不包括其后续 `preview` 修改 |
| [SHA256SUMS.txt](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.6/SHA256SUMS.txt) | 对应源码 ZIP 的 SHA-256 清单；不是历史安装包的校验表 |
| [Release-Notes-zh-CN.md](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.6/Release-Notes-zh-CN.md) | 本次源码发布、迁移警告及验证边界 |

## 历史源码附件：2.0.5 架构优化版

此版本是旧 PG 单库历史附件，不包含后来提交的健康采样整合或 MySQL 双库实现。

| 附件 | 用途 |
| --- | --- |
| [Suining-2.0.5-optimized.1-source.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.5-optimized.1/Suining-2.0.5-optimized.1-source.zip) | 完整历史源码，含字体 |
| [Suining-2.0.5-optimized.1-no-fonts-source.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.5-optimized.1/Suining-2.0.5-optimized.1-no-fonts-source.zip) | 无字体历史源码；复现完整界面需核对字体资源 |
| [无字体版说明](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.5-optimized.1/NO_FONTS_README_zh-CN.md) | 无字体附件的使用与资源说明 |
| [完整源码 SHA-256](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.5-optimized.1/SHA256SUMS.txt) · [无字体源码 SHA-256](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.5-optimized.1/SHA256SUMS-no-fonts.txt) | 必须按对应文件核验，不混用清单 |

## 历史 Windows 与 Linux 附件：2.0.2

以下文件仍属于 2.0.2；同一 Release 中的 Linux PG 管理工具版本是独立的 1.0.0。Windows PG 单库程序不支持当前 `SUINING_MYSQL_DSN` 主库配置。

| 附件 / File | 大小 / Size | 用途与限制 / Scope |
| --- | --- | --- |
| [Suining-Setup-2.0.2-win-x64.exe](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Setup-2.0.2-win-x64.exe) | 28.82 MiB | 历史 Windows 安装版；缺少 WebView2 时联网补装 |
| [Suining-Portable-2.0.2-win-x64.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Portable-2.0.2-win-x64.zip) | 30.69 MiB | 历史免安装版；完整解压并保留两个 EXE，需要已有 WebView2 |
| [Suining-Source-2.0.2-preview.1.zip](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Source-2.0.2-preview.1.zip) | 27.05 MiB | 该历史版本的源码附件；不是当前 preview 源码 |
| [Suining-Linux-amd64.tar.gz](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Suining-Linux-amd64.tar.gz) | 31.49 MiB | 用户提供的 Go 服务＋浏览器历史软件包；兼容性未确认 |
| [suining-db-manager_1.0.0_amd64.deb](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/suining-db-manager_1.0.0_amd64.deb) | 5.44 MiB | 独立 PG 管理工具，组件版本 1.0.0；不包含数据库服务 |

相关附件：[SHA-256 清单](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/SHA256SUMS.txt) · [当时的中文说明](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/Release-Notes-zh-CN.md) · [来源清单](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/release-manifest.json) · [当时的验证记录](https://github.com/WatchTestCs/smart-elderly-watch-management/releases/download/v2.0.2-preview.1/verification.json)。

Linux 包的运行依赖和已知兼容性限制单独记录在 [Linux 说明](LINUX.md)。原始 RAR、早期安装包与历史源码保留各自来源，不能通过重命名成为当前 2.1.0。

## 如何核对文件

1. 同时确认 Release 标签、文件名和交付类型。**2.1.0 提供三类交付，历史 v2.0.6 仅提供源码**；GitHub 自动生成的 Source code ZIP 与维护者上传的命名源码包可能有不同目录和文件，不应混用摘要。
2. 在同一 Release 下载对应 `SHA256SUMS.txt`，对文件计算 SHA-256；完整源码和无字体版分别使用各自清单。
3. 摘要匹配只证明下载字节符合清单，不证明该软件已通过当前业务、硬件或生产环境验收。

```powershell
Get-FileHash -Algorithm SHA256 -LiteralPath '.\实际下载的文件名'
```

## 数据与升级

旧 PG 业务库到当前 MySQL 需要显式离线导入，安装软件不会自动迁移。工具兼容 PG 0006／0007／0008，先预检查、再导入空目标；图片单独复制。配置与切换步骤见 [双库指南](DUAL_DATABASES.md)。旧 Java/H2、早期 Linux 附件不在自动导入承诺内。

当前源码的验证与旧本地安装包的区别见 [整合记录](INTEGRATION_2.0.6.md)。下载指引不代替数据备份，也不把历史附件的测试结果移用到当前版本。

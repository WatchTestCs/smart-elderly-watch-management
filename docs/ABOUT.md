# 岁宁 · About 文案与项目定位

本文件维护 `preview` 的项目简介、主页与 Topics 建议，供 README、发布说明和仓库 About 使用。文案对应 2.0.6 整合源码；修改本文件本身不会自动更改 GitHub 仓库级设置。

## 中文简介

岁宁｜面向照护机构与家属的智慧养老管理软件。Tauri 2 / Rust + Go + Vue 3 / Vite，MySQL 核心业务主库＋PostgreSQL 异步分析；支持长者档案、手表接入、健康采样、SOS 处置与独立复核。

## English description

Suining is an elder-care management application built with Tauri 2/Rust, Go and Vue 3/Vite. MySQL owns core transactions; PostgreSQL provides asynchronous analytics. It supports resident profiles, wearables, health samples and SOS care with independent verification.

## 主页

[preview 项目首页](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/preview)

当前源码和文档在 preview；源码下载、Windows 历史软件与 Linux 附件分别见 [附件索引](DOWNLOADS.md)。主页指向 preview 不意味着改动默认分支。

## Topics

`elder-care`、`tauri`、`rust`、`golang`、`vue3`、`vite`、`mysql`、`postgresql`、`desktop-app`、`windows`、`iot`、`health-data`

## 更完整的产品说明

岁宁将长者、监护人与家属档案、设备绑定、健康及位置记录、求助告警、照护履历和公告互动放在同一工作空间。主管、护理员和审计员在照护模块内承担不同职责；另一名主管独立复核后关闭任务，家属访问关联家人的记录。

桌面采用 Tauri 2、Rust 和 Windows WebView2，Vue 负责页面交互，Go 提供内嵌静态资源、业务 API、设备鉴权与后台作业。MySQL 承担全部核心事务和持久化队列，PG 接收可重建的分析投影；PG 故障不切换业务主库。前端不直连数据库，不持有数据库凭据或桌面初始化令牌。

## 对外描述的版本边界

| 内容 | 准确表述 |
| --- | --- |
| 当前源码 | 2.0.6 preview，MySQL 主库＋PG 分析，含健康采样整合 |
| 公开源码附件 | 截至 2026-10-19 为 2.0.5-optimized.1；它是旧 PG 单库源码 |
| 公开 Windows 二进制 | 仍为历史 2.0.2，不包含 2.0.6 双库与健康采样整合 |
| 本地旧 2.0.6 打包 | 未包含后续健康采样整合，不能代替当前源码的对应构建 |
| Linux 软件与管理工具 | 历史独立附件，未确认与当前源码、迁移和修复一致 |
| 长期值守 | 需要常驻后台、通知网关与人工响应流程；本地桌面关闭会停止其 Go 子进程 |

当前技术栈不再标注为 Spring Boot、Vue 2、Electron 或内置 H2。已有 Go 与浏览器验收不能表述为真实手表、短信网关、生产迁移或原生安装验收。具体证据见 [2.0.6 整合记录](INTEGRATION_2.0.6.md)。

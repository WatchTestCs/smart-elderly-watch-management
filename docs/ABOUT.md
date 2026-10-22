# 岁宁 · About 文案与项目定位

本文件维护 `preview` 的项目简介、主页与 Topics 建议，供 README 和仓库 About 使用。About 用一句话说明服务对象与核心用途；版本、技术栈、安装和验证细节由项目首页及专门文档维护。修改本文件本身不会自动更改 GitHub 仓库级设置。

## 中文简介

岁宁｜面向照护机构与家属的开源养老管理平台，连接智能手表、健康记录与照护协作。

## English description

Suining is an open-source elder-care platform for care organizations and families, connecting smart watches, health records and coordinated care.

## 主页

[preview 项目首页](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/preview)

当前开发源码和文档在 `preview`；2.1.0 安装版、免安装版、固定源码及历史附件见 [下载索引](DOWNLOADS.md)。主页指向 `preview` 不意味着改动默认分支。

## Topics

`elder-care`、`tauri`、`rust`、`golang`、`vue3`、`vite`、`mysql`、`postgresql`、`desktop-app`、`windows`、`iot`、`health-data`

## 更完整的产品说明

岁宁将长者、监护人与家属档案、设备绑定、健康及位置记录、求助告警、照护履历和公告互动放在同一工作空间。主管、护理员和审计员在照护模块内承担不同职责；另一名主管独立复核后关闭任务，家属访问关联家人的记录。

桌面采用 Tauri 2、Rust 和 Windows WebView2，Vue 负责页面交互，Go 提供内嵌静态资源、业务 API、设备鉴权与后台作业。MySQL 承担全部核心事务和持久化队列，PG 接收可重建的分析投影；PG 故障不切换业务主库。前端不直连数据库，不持有数据库凭据或桌面初始化令牌。

## 对外描述的版本边界

| 内容 | 准确表述 |
| --- | --- |
| 当前源码 | `preview` 持续维护；MySQL 主库＋PG 异步分析，含发布后的配置诊断和 SQL 翻译修复 |
| 公开源码附件 | v2.1.0 固定发布快照，与同版 Windows 程序对应，不等于持续更新的分支源码 |
| 公开 Windows 二进制 | v2.1.0 安装版及免安装版，未签名；实际检查以随包来源清单为准 |
| Linux 软件与管理工具 | 历史独立附件，未确认与当前源码、迁移和修复一致 |
| 长期值守 | 需要常驻后台、通知网关与人工响应流程；本地桌面关闭会停止其 Go 子进程 |

About 不写固定版本号或堆叠技术名词，避免简介随发布失效；技术架构见[项目首页](../README.md#系统架构)。已有 Go、浏览器和打包检查不能替代真实手表、短信网关、生产迁移或完整原生窗口交互验收。当前发布证据见 [2.1.0 说明](releases/2.1.0.md)与随包 `release-manifest.json`。

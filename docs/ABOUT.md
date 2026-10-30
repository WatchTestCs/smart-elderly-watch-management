# 岁宁 · 仓库简介与项目定位

本文件汇总岁宁项目的简短介绍、推荐主页链接与 GitHub Topics 标签，方便在仓库设置与对外文档中保持一致。

## 中文简介

岁宁：面向照护机构与家属的开源养老管理平台，连接智能手表、健康记录与照护协作。

## English description

Suining is an open-source elder-care platform for care organizations and families, connecting smart watches, health records, and coordinated care.

## 仓库主页与 Topics

- 推荐主页链接：`https://github.com/WatchTestCs/smart-elderly-watch-management/tree/preview`
- 推荐 Topics：`elder-care`、`tauri`、`rust`、`golang`、`vue3`、`vite`、`mysql`、`postgresql`、`desktop-app`、`windows`、`iot`、`health-monitoring`

## 产品定位说明

岁宁将长者、监护人与家属档案、手表设备绑定、健康与位置记录、SOS 求助告警、照护工单履历和公告互动整合在同一工作空间内。主管、护理员和审计员在照护模块内按各自岗位权限完成接单处置与独立复核留痕。

Windows 桌面端采用 Tauri 2、Rust 与 WebView2 构建，由 Vue 3 负责前端交互，Go 提供内嵌静态资源、业务 API、设备鉴权与后台作业。MySQL 承担全部实时业务读写，PostgreSQL 异步接收事件投影以支持离线报表与趋势分析。

## 版本与交付物说明

| 交付物类型 | 对应范围 |
| --- | --- |
| `preview` 分支源码 | 当前持续演进的开发分支，采用 MySQL 主库 + PostgreSQL 异步分析双库架构 |
| v2.1.0 源码归档 (`Source ZIP`) | 对应 v2.1.0 发布标签的固定源码快照，与同版本 Windows 安装包完全匹配 |
| v2.1.0 Windows 客户端 | 提供安装版（Setup）与免安装版（Portable ZIP），当前未附带商业代码签名 |
| 历史 Linux 附件 | 早期版本留存的独立辅助工具归档，不包含在当前源码的持续构建矩阵中 |

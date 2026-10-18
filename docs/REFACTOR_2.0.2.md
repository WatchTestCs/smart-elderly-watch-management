# 2.0.2 源码拆分与清理边界

本轮只按业务职责拆分已有 Go 处理器和 Vue Options API 视图；原有外部协议、数据库表、迁移脚本、HTTP 路由、状态字段与文件访问语义保持不变。此文件是重构后的维护指引，不代表自动证明运行行为完全等价。

## Go 处理器边界

- 老人档案：`handler_old_person.go`；设备通用解析和数据访问：`handler_device_parse.go`、`handler_device_repository.go`。
- 设备心跳/绑定协议：`handler_device_protocol.go`；遥测、上报、SOS：`handler_device_telemetry.go`。
- 设备列表与绑定码：`handler_device_admin.go`；设备创建/密钥更新/解绑/删除：`handler_device_inventory.go`。
- 告警解释与查询：`handler_device_alerts.go`；待处置记录与人工确认：`handler_device_alert_actions.go`。
- 文章、展示图、图片 I/O、短信：`handler_social_text.go`、`handler_social_pictures.go`、`handler_social_images.go`、`handler_social_sms.go`。
- 健康记录、步数与告警：`handler_vitals_ratepre.go`、`handler_vitals_measurements.go`、`handler_vitals_steps.go`。
- 家属管理与邀请码：`handler_family.go`、`handler_family_invites.go`、`handler_family_invite_join.go`。

各文件仍属 `package handlers`。分拆不引入跨 package 依赖；新逻辑应优先进入对应职责文件，不应把公共数据校验与数据库 I/O 混入 HTTP 业务函数。

## Vue 和样式边界

`App.vue` 仅保留根容器与生命周期。全局 CSS 顺序固定为 foundation → components → family/responsive，禁止随意改变，以免影响级联。

较长页面使用外部 Options API `*.logic.js` 文件；`device.logic.js` 将 methods 依列表、告警、刷新、设备管理分别组合。状态、生命周期、页面模板及 API 接口均维持原名称。不要在原文件与拆出的方法模块内重复定义同名 method。

`fonts.css` 为字体子集规则，按其生成/引用规则整体保留；`commercial.css` 采用压缩的短行 CSS，不能仅根据字节数认定必须拆分。

## 保留与验收

`handler_old.go` 是有效路由入口，不是历史废文件。嵌入式 Web 目录占位文件、全部 SQL 迁移、品牌字体许可、发布脚本、来源记录和运行资源均应保留。不要基于名称中包含 old、静态搜索未命中或缺少本地调用而删除兼容接口、反射/协议入口。

合并门槛：`gofmt`、`go test -race`、`go build`、`go vet`、前端锁文件安装及生产构建、UI Quality、Security 全部通过，并审查 PR 文件清单。更高风险的设备/短信/数据库端到端验证须单独记录，不能以静态检查替代。

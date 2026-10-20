# 平台改造本地验证记录

日期：2026-10-20。基点 `d6755ecc66efb11325a41e6d642ebcc0565fcafc`，包含本次未发布的增量源码。

## 执行范围

| 检查 | 本地结果与边界 |
|---|---|
| Go `test ./...`，提供真实 TEST_MYSQL_DSN | 116 个测试/子测试 pass，4 个 skip，无 fail；所有 MySQL 测试使用独立随机库 |
| 最后会话/2FA边界调整 | 数据库会话与 TOTP 2 项定向复核，无 fail，覆盖启用/关闭并发代际及恢复码重放 |
| 最后观测边界调整 | observability/config 包复核，检查未知方法/路由标签不扩散及请求 ID 防伪造 |
| Go `build ./...` / `vet ./...` / gofmt | 构建与 vet 返回 0；gofmt 无待格式化文件 |
| 前端 Vitest | 3 文件、7 单测；客户页业务失败、表单复制、分页重试、2FA确认及版本化接口 CSRF |
| 前端 `vue-tsc --noEmit` / Vite build | 返回 0；仅已迁移子集作严格 TS 检查；保留现有大 chunk 警告 |
| OpenAPI 生成一致性 | 123 路径，`--check` 返回 0；旧 DTO 未全量精确定义 |
| Python 脚本语法 | loadtest/generate-openapi/start-platform/bootstrap-admin 可编译 |
| Compose | `start-platform.py --validate` 返回 0；密钥与 env 未入库 |
| JSON 日志 | 实际服务产生结构化请求日志；首轮 13,989 条 JSON 记录可解析，检查无正文/手机号/secret 字段 |
| DDL | 0005 正向迁移、0005→0004回滚及重新迁移在临时 MySQL 库执行；历史照护记录、健康时钟保留 |

4 个跳过项是：`TestDualDatabaseRecoveryAndReplay` 与 `TestLegacyTransferIsCheckedAndAtomic` 的 PG0006/PG0007/PG0008，原因是本次没有配置 PostgreSQL 测试服务。这里不能代替双库恢复验证。

规则用模拟数据覆盖：基线样本不足、同源统计、连续不同观测、重复扫描、更正恢复、夜间累计步数和区间活动、计数器重置、并发离线去重。通知覆盖超过 100 条入队不丢、事务回滚不外漏、5 次失败语音降级、无网关站内渠道、过期租约不覆盖新结果。安全覆盖密文不直接暴露、AES-GCM 随机性/篡改检测、RFC TOTP 向量、防重放和跨副本撤销。

未执行：Docker 镜像构建及整套容器启动（本机 Docker Linux engine 管道不存在）、真实短信/语音/设备联调、生产数据导入、浏览器视觉验收、Windows 安装包发布、Go race 检查。DOM 单测不是浏览器验收；MySQL 8.0 的本机结果也不等同于 Compose MySQL 8.4 的运行记录。GitHub Actions 状态未查询。

## 模拟写入压测

环境：Windows、本机 AMD Ryzen AI 9 H 365（10 核/20 线程），约 31.1 GiB RAM；Go 1.27.2，Python 标准库 HTTP 客户端；MySQL 8.0.46，回环 TCP，独立测试库。单应用实例、所有服务端后台 worker 启用、JSON 请求日志写文件；不配置 PG 分析副本和外部通知网关。

场景：同一已绑定模拟设备携带真实服务端校验凭据，`POST /restful/device/healthBatch`，每次一个唯一心率样本。包含认证、绑定锁、规范化、健康写入/变更日志、事务提交及设备在线时间更新；客户端复用连接。每轮检测 HTTP 200 与业务 returnCode=1111，再按运行 ID 查询真实样本行数。数据库已有前序模拟记录，不是空库；每轮只有一台设备，是共享写入热点。

| 并发 | 秒 | 请求数 | req/s | P50 ms | P95 ms | P99 ms | max ms | 错误 | 核对写入行数 |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 30.001 | 1710 | 57.0 | 14.681 | 26.368 | 43.672 | 726.755 | 0 | 1710 |
| 8 | 30.015 | 11458 | 381.74 | 17.195 | 35.724 | 92.145 | 805.541 | 0 | 11458 |

机器同时执行 Go 集成测试的附加轮次曾测得单并发 93.98 req/s、最大时延 1512.477 ms；环境争用会明显影响结果。上表采用集成测试结束后独立执行的轮次，不能拿峰值代替稳定容量。首轮探索数据为单并发 225.14 req/s、8 并发 473.68 req/s，均 0 业务错误，也不作为生产承诺。

未测：多设备、多租户、跨主机扩容、长时间稳态、真实网络/HTTPS、通知网关吞吐、故障重启时持续压测、生产峰值、CPU/内存曲线。`health_clock` 与同设备事务锁会限制写入并行度；本次没有据此改动已有一致性方案。复现脚本 `scripts/loadtest.py`；原始控制轮结果在同目录 `platform-load-c1.json` / `platform-load-c8.json`。

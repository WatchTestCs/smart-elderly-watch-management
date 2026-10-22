# Go 1.23 性能优化与 200,000 条/秒验收边界

日期：2026-10-20。测试环境：Linux amd64、Go 1.23.2、Intel Xeon Platinum 8573C。`GOMAXPROCS=1` 的基准用于控制调度差异；不同运行仍存在测量波动。

## 已修改

1. `backend/internal/health/batch_sql.go`：将反复构建的批量占位符 SQL 模板改为线程安全、按需、固定上界缓存（1/4/14 列、最多 200 行）。`store.go` 调用不变，生成的 SQL 文本保持原样；不会改变批次大小、事务语义或持久化行为。
2. `backend/internal/health/batch_sql_test.go`：覆盖模板文本、无缓存回退、200 行上界和多协程并发一致性。
3. `backend/go.mod`：将声明的 Go 语言最低版本设为 1.23.0；替换了本包内仅支持较新 Go 的 `testing.B.Loop()` 用法。**由于缺少三方依赖/数据库，此项只完成 Go 1.23 核心代码验证，完整工程的 Go 1.23 依赖兼容性尚未证实。**
4. `scripts/go123-core-benchmark.py`：离线将直接使用的生产规则、样本标准化及 SQL 构造源码放入临时小模块，用 Go 1.23 测试并测量。不会伪装成数据库压测。
5. `scripts/verify-health-scale.py` 与对应 7 项 Python 测试：必须提供同一轮的独立数据库 `health_sample`、`health_change` 行数才能判断 20 万条/秒是否达成；只收 HTTP ACK 的结果不合格。

原有通知 Outbox 多工作者代码保留，未提高默认并发，也没有关闭持久化、校验或可靠性措施换取虚假吞吐。

## 实测

| 对象 | 优化前（原源码） | 优化后（本源码） | 说明 |
| --- | --- | --- | --- |
| SQL 模板 200×14 | 61–65 µs/op、18,432 B/op | 3.6–5.9 ns/op、0 B/op | 只测字符串模板，不含 MySQL 执行 |
| SQL 模板 200×4 | 16–17 µs/op、6,144 B/op | 3.0–3.3 ns/op、0 B/op | 同上 |
| 规则计算 | 约 3.9–4.5 µs/op、54 alloc/op | 未修改该算法 | 单核运算约 22–25 万次/秒，不含数据库 |
| 标准化 + JSON + 身份 SHA-256 | 约 2.57–2.94 µs/op | 维持原实现 | 约 34–39 万条/秒，非端到端 |
| **端到端 MySQL 持久化** | **历史最佳：约 13,058 样本/秒** | **本环境无法运行 MySQL，未测** | **没有达到、也不能宣称达到 200,000** |

Go 1.23 隔离运行：规则/健康测试通过；两者的 `-race` 检查通过；缓存并发回归通过；Python 验收判定 7/7 通过。完整 `go test ./...` 已尝试，但因第三方模块未在环境中缓存且不能联网而无法编译；没有把跳过视为通过。未进行真实 MySQL 8.4、通知网关、弱网/中断恢复及四实例重测。

## 后续如何验收 200,000 条/秒（避免数学假象）

使用隔离的 MySQL 8.4、至少 4 个应用实例、独立负载机与网关模拟服务，固定请求批次为 200 样本，使用现有 `scripts/cluster-loadtest.py`，增加并发、连接数和设备数时同时记录 CPU、MySQL p95/p99 SQL 延时、提交次数、锁等待、错误与重试。20 万样本/秒意味着至少 **1,000 次成功提交的 200 样本批次/秒**；测试需持续多轮，且不能用单实例速率相乘。

```bash
python scripts/cluster-loadtest.py --config isolated-fixture.json --output run.json --processes 8 --threads 32 --seconds 30 --batch-size 200
```

应在所有请求结束、事务完成后，用 `run.json` 的 `run_id` 替换下面的 `YOUR_RUN_ID`，独立查询测试数据库（**可能触发大表扫描，请在测试库执行**）：

```sql
SET @run = 'YOUR_RUN_ID';
SELECT count(*) AS persisted_samples FROM health_sample
 WHERE source='cluster-benchmark'
 AND JSON_UNQUOTE(JSON_EXTRACT(payload,'$.recordId')) LIKE CONCAT(@run,'-%');
SELECT count(*) AS persisted_changes FROM health_change
 WHERE JSON_UNQUOTE(JSON_EXTRACT(payload,'$.sample.recordId')) LIKE CONCAT(@run,'-%');
```

把查询结果手动写入 `db-evidence.json`（独立记录真实查询结果，不能复制 ACK 数字）：

```json
{"run_id":"YOUR_RUN_ID","scope":"recordId prefix YOUR_RUN_ID-","persisted_samples":0,"persisted_changes":0}
```

```bash
python scripts/verify-health-scale.py --report run.json --db-evidence db-evidence.json --target 200000
```

仅在无 HTTP 错误、HTTP ACK 与同一轮的两个数据库计数相同且实际速率至少 200,000 时，脚本才返回成功。它不能证明数据库证据的来源真实性，仍需由操作者保留 SQL 原始输出和服务日志。通知系统还需独立核对 Outbox 行数、唯一发送 ACK、重试与故障恢复；短信网关接受不等于短信送达。

**结论：** 已完成 Go 1.23 核心测试和 SQL 生成热点优化；20 万条/秒端到端目标仍未验收。无法从该环境向 MySQL 服务证明目标已实现。

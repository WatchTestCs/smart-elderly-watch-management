# 后端性能迭代报告

执行日期：2026-10-24 至 2026-10-25。基线：`preview@daac58a9d20420cc4bdf71675be64bcc20cc8e49`，交付前已重新核对远端仍为该提交。原始实验在独立 preview 工作副本中完成；本报告记录固定基线和候选实现的本地结果，不将其当作 CI 或生产部署证明。后续复审与发布验证见 [REVIEW.md](REVIEW.md)。

**结果：已登录读取的真实 HTTP p50 降低约 34%–68%，1 万条未缓存统计降低约 39%。登录基本持平；设备写入未作加速结论。不能承诺所有输入和所有分位数都更快。**

## 最终保留的实现

1. 参考 [SCS](https://github.com/alexedwards/scs) 与 [Gorilla Sessions](https://github.com/gorilla/sessions) 的请求内加载模式，合并 session/MFA generation 查询，并按 Store、cookie、有效期在单次请求内复用已验证会话。从源码路径看，受保护 POST 的认证读取由 4 条 SELECT 降为 1 条；这个查询数来自代码分析，不是独立 SQL 计数实验。没有增加跨请求 TTL 缓存。新请求和直接 Get 仍查询数据库；已开始的请求使用已验证快照。密钥校验和 bcrypt cost=12 不变。
2. 参考 [Spring Data](https://github.com/spring-projects/spring-data-commons/blob/4.0.0/src/main/java/org/springframework/data/support/PageableExecutionUtils.java)、[Django](https://github.com/django/django/blob/5.2/django/core/paginator.py) 和数据库优化器文档。客户/公告采用独立精确 COUNT LEFT JOIN 有界页面，保留空页总数；家属保留原始分页查询，在非空尾页和空首页推导总数，其他情况仍精确 COUNT。
3. Aggregate 使用请求内样本指针和精确日历元组，减少大结构复制与日期字符串分配。活动扫描复用时区、日期和 Clock 分解结果；Location 改变时重新计算上次日期。原步数整数守恒、排序、过滤、错误检查和日界语义保留。原有小数日累计求和仍存在 map 遍历顺序带来的浮点不确定性，本次未把它改为新的确定性契约。
4. 直接复用 MIT 授权的 [goccy/go-json v0.11.2](https://github.com/goccy/go-json/tree/v0.11.2)，只替换数据库存储 Sample 的解码。请求解析、会话解析、规范化序列化和缓存编码仍用 encoding/json。依赖固定、校验和验证通过，授权全文见 `docs/licenses/goccy-go-json-LICENSE.txt`。兼容性回归覆盖 317 种输入的 fresh/seeded 共 634 次对照及数据库缓冲区所有权；这不是穷尽证明。

没有为了替换架构而迁移存储引擎。RocksDB/MyRocks、pgx 批处理及驱动参数复用均进行了适用性比较；现有部署的数据归属、事务确认、异步分析与接口契约继续保留。详细来源与许可证见 [OPEN_SOURCE_REFERENCES.md](OPEN_SOURCE_REFERENCES.md)、[JSON_DECODING.md](JSON_DECODING.md)。

## 真实 HTTP 最终对照

Windows amd64，Go 1.23.2，AMD Ryzen AI 9 H 365，GOMAXPROCS=2；独立 Docker MySQL 8.4.13，InnoDB buffer pool 256 MiB，业务池上限20，默认每连接语句缓存64。保留 innodb_flush_log_at_trx_commit=1、sync_binlog=1 和 binlog。后端通过本机端口访问 Docker Desktop 虚拟机中的数据库，客户端在同一 Windows 进程；不是独占硬件。

按基线 A3 → 最终 B3 → 最终 B4 → 基线 A4 的顺序执行。每次普通读取各300次×3轮，未缓存统计40次×3轮，登录20次（另预热3次，低于30次/10分钟限流）。下表每个数字为两次运行中6个轮次相应分位数的中位数；登录仅两组各20次。**不是合并全部样本计算的分位数，也没有统计显著性保证。** 单位 ms。

| 场景 | p50 基线→最终 | p50变化 | p95 基线→最终 | p99 基线→最终 |
|---|---:|---:|---:|---:|
| 已登录空请求 | 1.628 → 0.526 | -67.7% | 2.330 → 1.041 | 2.662 → 1.229 |
| 客户列表 | 2.846 → 1.599 | -43.8% | 3.687 → 2.244 | 4.120 → 2.775 |
| 设备列表 | 2.690 → 1.617 | -39.9% | 3.549 → 2.418 | 4.298 → 2.837 |
| 家属列表 | 2.379 → 1.340 | -43.7% | 3.069 → 2.066 | 3.391 → 2.342 |
| 变更流（100条） | 2.611 → 1.506 | -42.3% | 3.693 → 2.176 | 4.214 → 2.572 |
| 登录 | 189.493 → 188.363 | -0.6% | 216.764 → 196.214 | 229.750 → 200.941 |
| 通知列表（100条） | 2.392 → 1.229 | -48.6% | 3.194 → 1.982 | 3.625 → 2.438 |
| 老人列表 | 3.050 → 2.004 | -34.3% | 3.708 → 2.736 | 4.250 → 3.129 |
| 热统计 | 2.045 → 1.036 | -49.3% | 2.809 → 1.581 | 3.145 → 1.784 |
| 未缓存统计（1万条） | 37.483 → 22.817 | -39.1% | 46.375 → 24.361 | 49.178 → 24.806 |
| 公告列表 | 2.830 → 1.586 | -43.9% | 3.658 → 2.281 | 4.247 → 2.705 |

包括真实 socket、设备外的数据库会话认证、CSRF、角色校验、限流、指标和日志格式化；日志写 io.Discard。没有外网、TLS、代理、WebView、磁盘日志、生产外层响应处理或后台规则/分析/通知竞争。读取为串行闭环，不能推导生产并发容量。列表30条合成数据、每页20条；通知与变更流各返回100条；统计持久化10000条心率样本，响应 count=10000、mean=72。应用统计缓存对 hot 明确开启、对 uncached 关闭；数据库缓冲池与 JSON 类型解码缓存均预热，**不是进程冷启动或冷磁盘测试**。

每次响应均校验 HTTP/业务码、列表长度和统计结果；检查种子样本/事件数量、成功登录审计计数，并验证跨实例撤销在新请求上立即生效。原始 JSON 中 fixture 的 commit 字段固定为基线，不作为候选版本证明；候选以外层运行标签和本次提交中的生产文件识别；[生产文件散列](evidence/production-source-hashes.json)确认复审后的生产 Go 与模块文件和实测版本逐字节相同，后续修正只涉及测试及文档。进程分配含客户端和响应验证，不能当成服务器单独分配。

## 纯计算与分配

相同基准复制到两版；A1/B1/B2/A2，每项200ms/轮，各版本两轮。只测已解码样本，排除数据库和 JSON；不是接口加速倍数。单位 μs/op 与 B/op。

| 基准 | 时间 基线→最终 | 变化 | 分配 基线→最终 |
|---|---:|---:|---:|
| ActivityWindow/daily_total/1000/stream | 48.31 → 32.86 | -32.0% | 0 → 0 |
| ActivityWindow/interval/1000/stream | 67.65 → 60.48 | -10.6% | 0 → 0 |
| Aggregate/daily_total/1000 | 178.04 → 51.76 | -70.9% | 810584 → 17552 |
| Aggregate/instant/1000 | 80.45 → 9.54 | -88.1% | 794344 → 17552 |
| Aggregate/interval/1000 | 125.40 → 54.76 | -56.3% | 794656 → 17608 |
| Aggregate/sleep/1000 | 135.50 → 74.37 | -45.1% | 1088472 → 311680 |
| Aggregate/sparse/1000 | 4.64 → 1.02 | -78.0% | 7912 → 272 |

独立解码实验中，1000条混合样本 encoding/json 约2.406ms，go-json约0.543ms（-77.4%）；完整接口收益以上述 HTTP 表为准。详细原始实验包含不利结果，见 [解码器实验记录](evidence/decoder-experiment.txt)。

## 分页规模、回退与淘汰

处理器微基准不含认证/真实HTTP；100/10000条、每页20条、公告正文512字节，覆盖首页、末页、筛选尾页及越界页；最终 A3/B3/B4/A4，各60次。它与上面 HTTP 小表端点不是同一计时范围。单位 ms/op。

| 场景 | 基线→最终 | 变化 |
|---|---:|---:|
| 100/customer/first | 1.193 → 0.929 | -22.1% |
| 100/customer/last | 1.309 → 0.940 | -28.2% |
| 100/customer/out_of_range | 1.165 → 0.889 | -23.6% |
| 100/customer/selective | 0.797 → 0.455 | -42.9% |
| 100/family/first | 0.895 → 0.941 | +5.2% |
| 100/family/last | 0.914 → 0.900 | -1.5% |
| 100/family/out_of_range | 0.830 → 0.882 | +6.2% |
| 100/family/selective | 0.811 → 0.437 | -46.1% |
| 100/text/first | 1.624 → 1.062 | -34.6% |
| 100/text/last | 1.511 → 1.018 | -32.6% |
| 100/text/out_of_range | 1.409 → 0.989 | -29.8% |
| 100/text/selective | 0.948 → 0.481 | -49.2% |
| 10000/customer/first | 1.417 → 1.481 | +4.5% |
| 10000/customer/last | 2.582 → 2.243 | -13.1% |
| 10000/customer/out_of_range | 2.552 → 2.161 | -15.3% |
| 10000/customer/selective | 3.111 → 2.435 | -21.7% |
| 10000/family/first | 8.248 → 8.182 | -0.8% |
| 10000/family/last | 8.839 → 8.799 | -0.5% |
| 10000/family/out_of_range | 8.917 → 8.703 | -2.4% |
| 10000/family/selective | 1.184 → 0.658 | -44.4% |
| 10000/text/first | 6.455 → 6.010 | -6.9% |
| 10000/text/last | 15.046 → 15.306 | +1.7% |
| 10000/text/out_of_range | 14.624 → 14.927 | +2.1% |
| 10000/text/selective | 3.917 → 3.265 | -16.6% |

小表家属首页/越界页在这组微基准中约有5%–6%回退，补充三方案对照方向又有所变化，因此不宣称每个输入都更快，也不直接归因于环境噪声。全部数据保留。

- 第一版 CROSS JOIN 在10000条家属筛选中约+76%，客户越界约+27%，已撤回。
- 第三版家属 scalar COUNT 在补充对照中，10000条筛选约+101%；普通首页虽更快，仍因严重筛选回退淘汰，**恢复第二版**。恢复文件 SHA-256 为 `df19536e688e17bb39b923947e65cc69f482a7e24eae75de7c79e944612ccf59`，与最终 B3/B4 测量使用的第二版一致。
- Sonic v1.15.4 ConfigStd 本地可运行，但输入 `{"value":-0}` 被解为 +0，改变浮点位与标准序列化结果，未采用。保留失败日志，未放宽比较标准。
- Easyjson 已研究但未基准测试；生成代码的维护成本没有得到本项目端到端收益证明。

## 写入对照

使用已有真实 healthBatch fixture，各300请求一次；包含设备认证、锁和真实提交，直接挂载 handler，**不含读取测试的外层指标/限流链路**。每版仅一次、未交替，因此只作正确性与时延控制，不能证明写入改进或可靠回退。

| 版本/场景 | p50 ms | p95 ms | p99 ms | 请求/s |
|---|---:|---:|---:|---:|
| baseline / 1条×1客户端 | 8.823 | 11.257 | 12.917 | 110.05 |
| baseline / 200条×4客户端 | 89.849 | 112.141 | 171.553 | 42.47 |
| final / 1条×1客户端 | 9.354 | 12.798 | 15.062 | 102.21 |
| final / 200条×4客户端 | 72.875 | 98.825 | 168.657 | 51.90 |

所有接受数、零重复和持久化事件数核对通过。此次未修改写入生产路径，也没有 fsync trace，不能称写入已达硬件地板。

## 原始实验验证记录

- Windows完整后端短测实际执行：393个测试/子测试通过、11个测试跳过、4个失败。3个失败是连接串缺少显式专用库名，改用隔离 control 库后 internal/db 全包重跑通过；1个是旧 analytics backlog 测试在 PG 已写入而 MySQL 确认尚未完成时观察到250条剩余，internal/analytics 全包重跑通过，基线专项3次也通过。原实验保留了此失败；提交前复审已修正该测试的确认时序，见 [REVIEW.md](REVIEW.md)。**不将原始失败改写为通过**。
- 最终新增分页/JSON/DST回归通过；分页在真实 MySQL8.4 与 PostgreSQL16 均通过。
- Linux amd64 Go1.23.12，六个相关包的定向 -race -short 检查通过；不是所有包的完整race验收。包括 session/auth/handlers/health/rules/cmd/server。
- go vet ./...、go build ./...、go mod verify 通过。第三版淘汰后恢复相同已测第二版文件；恢复后的非DB重点测试通过。
- 最终源码另经独立只读审查，未发现可操作的正确性问题；这不替代上述执行证据。未运行生产部署、真实设备、开放到达率、长时间耐久或最终依赖在所有Go版本/架构上的验证。

## 仓库内证据与复现

公开测量数据见 [evidence/README.md](evidence/README.md)，汇总见 [results-summary.json](results-summary.json)。[四次 HTTP 原始报告](evidence/http-runs.jsonl)包括每次时延；[微基准](evidence/microbenchmarks.txt)、[淘汰方案](evidence/rejected-pagination.txt)、[解码器对照](evidence/decoder-experiment.txt)保留所有方向的数据。原始文件散列见 [source-hashes.json](evidence/source-hashes.json)。不依赖未上传的本地路径或附件来证明上表数字。

在基线 daac58a9d20420cc4bdf71675be64bcc20cc8e49 和候选代码两个独立 checkout 上执行相同 fixture。将候选的 `智慧养老手表管理系统/backend/internal/handlers/read_latency_ab_test.go` 原样复制到基线相同位置；两个版本均从 backend 目录执行：

```powershell
$env:TEST_MYSQL_DSN = '<专用隔离 MySQL DSN，须含显式库名并允许创建/删除临时库>'
$env:GOMAXPROCS = '2'
$env:RUN_READ_LATENCY_AUDIT = '1'
$env:PERF_AB_VARIANT = 'baseline' # 候选改为 candidate
$env:READ_LATENCY_RESULT_PATH = '<本轮报告的绝对路径.json>'
go test ./internal/handlers -run '^TestReadLatencyAudit$' -v -count=1 -timeout 15m
```

按基线/候选/候选/基线交替，逐轮设置唯一结果路径。fixture 创建和删除独立临时库，校验响应和持久化数量；不要使用生产 DSN。它明确启用 hot 统计缓存，并另测关闭缓存的同一查询；不会启用跨请求会话缓存。普通读取每轮300次×3、未缓存统计40次×3、登录20次另预热3次，保留生产登录限流。

分页基准把候选 `list_pagination_benchmark_test.go` 同样复制到基线，执行 `go test ./internal/handlers -run '^$' -bench '^BenchmarkListPagination$' -benchmem -benchtime=60x -count=1`。纯计算对照使用两版相同 `BenchmarkAggregate`/`BenchmarkActivityWindow` 文件，执行 `go test ./internal/health ./internal/rules -run '^$' -bench '^Benchmark(Aggregate|ActivityWindow)$' -benchmem -benchtime=200ms -count=1`。需要逐版比较相同输入和基准，不能把 reference oracle 的不同内部实现直接当成生产加速比。

新增兼容性回归：`go test ./internal/handlers -run '^TestStoredSampleDecoder' -count=1`。真实双库回归与 race 仍需配置相应 TEST_MYSQL_DSN/TEST_DATABASE_URL；默认跳过的数据库测试不算通过。

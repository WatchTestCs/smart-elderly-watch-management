# PR #30 历史描述快照

当前轮开始时的描述，对应 93d609b；以下是历史实验和全部原始数据，不代表后续提交的验收。

## 当前延迟优化与验收提交

提交 **93d609b10ec44b7245cfcde3173bb8da7e86512e**，基于 e39e14b 继续降低延迟。此精确提交的双库 CI、必需检查和全部 A/B 均通过。

- MySQL 物理连接各自持有最多 64 个 prepared statement，减少重复解析、插值缓冲和 prepare 往返；FIFO 有界，不增加共享缓存锁或额外池 checkout。专用健康探测与 PostgreSQL 保持原路径。
- 句柄容量先释放后替换，避免服务器额度已满时无法创建新句柄。参数转换、事务隔离、取消与连接存活接口原样转发；参数数量执行前检查，执行错误不重放。只有 prepare 阶段 1295 可回退原驱动。SET/USE/DDL 等保守清理句柄，避免旧解析状态。
- 明确代价：20 条池连接默认最多 1280 个服务器句柄/副本；首次请求、SQL 形状抖动可能变慢。环境变量 MYSQL_STMT_CACHE_SIZE=0 回到上一版；允许 0..128。
- 新增真实 HTTP 四种配置（1/200 条、1/4 客户端），同设备并发覆盖锁排队，各三轮×500 请求，测 p50/p95/p99/max，核对响应与持久化计数；继续 SaveBatch+commit 500 批对照。基线全部更新为 e39e14b。

[实现、维护约束与测量边界](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/perf/backend-hot-path-20261023/docs/performance/2026-10-23-latency)。本地 race -short/build/vet 通过，数据库本地跳过。最终 Go 1.23.2/1.27.2 的真实 MySQL/PostgreSQL race、build、vet、格式检查，以及必需 CI、前端、依赖审查、CodeQL Go/JS 和 A/B 均已通过。HTTP 测量包含 loopback 网络与完整健康处理器，不含全局中间件、外网 RTT、开放到达率排队或生产 SLA。

## 最终提交 93d609b 的真实延迟结果

[最终 A/B 运行 39921085461](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39921085461)，相对 **e39e14bf8500efbd74548ef012d244eaf8f320b1**；同 runner、MySQL 8.4、Go 1.23.2、GOMAXPROCS=2。每配置各三轮、每轮预热后 500 次请求，交替顺序，各自三轮中位数：

| HTTP 配置 | p50 基线→候选 ms | p95 基线→候选 ms | p99 基线→候选 ms |
| --- | ---: | ---: | ---: |
| 1 条、1 客户端 | 3.446→3.287（-4.6%） | 3.857→3.990（+3.4%） | 6.317→6.658（+5.4%） |
| 1 条、4 客户端 | 11.150→10.608（-4.9%） | 13.195→13.462（+2.0%） | 17.412→17.253（-0.9%） |
| 200 条、1 客户端 | 20.784→19.620（-5.6%） | 27.665→26.243（-5.1%） | 31.656→30.360（-4.1%） |
| 200 条、4 客户端 | 82.640→77.678（-6.0%） | 96.972→89.017（-8.2%） | 105.166→94.166（-10.5%） |

200 条 HTTP 请求分配串行 **1611637→1170358 B（-27.4%）**，4 并发 **1643374→1205029 B（-26.7%）**；GC 次数分别 **371→249（-32.9%）**、**506→364（-28.1%）**。小请求分配约 -8.6%。

独立 SaveBatch+commit：p50 **17.913→16.485 ms（-8.0%）**、p95 **25.879→23.072 ms（-10.8%）**、p99 **30.679→29.052 ms（-5.3%）**，吞吐 **10490.2→11527.0 样本/s（+9.9%）**；分配 **1113323→701863 B/批（-37.0%）**，GC **190→120（-36.8%）**。

**没有证明所有场景尾延迟稳定下降。** 最终的小请求串行 p95/p99 有回退；同代码主体的首轮（8e9553a，后续仅修复驱逐先释放句柄）也存在相反尾部方向，见下表。不同运行的绝对延迟差别较大，不能把候选最好结果挑出作为生产承诺，更不能声称已发挥全部硬件性能。所有成功轮次均零错误，响应及持久化数量校验通过。

| 首轮 HTTP 配置 | p99 基线→候选 ms |
| --- | ---: |
| 1 条、1 客户端 | 28.200→18.858（-33.1%） |
| 1 条、4 客户端 | 52.114→61.866（+18.7%） |
| 200 条、1 客户端 | 83.288→85.765（+3.0%） |
| 200 条、4 客户端 | 154.243→160.769（+4.2%） |

验收：[最终双库 CI](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39921091810)、[最终安全检查](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39921091820)。[首轮 A/B](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39920705132) 同样保留，不因尾部回退删除。

<details><summary>最终提交全部 30 轮原始 JSON（6 轮事务 + 24 轮 HTTP）</summary>

head_sha: 93d609b10ec44b7245cfcde3173bb8da7e86512e; baseline: e39e14bf8500efbd74548ef012d244eaf8f320b1; run_id: 39921085461

```json
[
{"allocated_bytes_per_batch":1114069,"allocations_per_batch":5534,"batches":500,"elapsed_ms":10664.541755,"errors":0,"gc_cycles":190,"gc_pause_ns":7903937,"go":"go1.23.2","gomaxprocs":2,"p50_ms":18.705734,"p95_ms":29.490397,"p99_ms":80.849271,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":9376.867970263764,"variant":"baseline"},
{"allocated_bytes_per_batch":701863,"allocations_per_batch":5932,"batches":500,"elapsed_ms":8675.308678,"errors":0,"gc_cycles":120,"gc_pause_ns":6334764,"go":"go1.23.2","gomaxprocs":2,"p50_ms":16.485231,"p95_ms":23.072375,"p99_ms":31.292585,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":11526.96736354676,"variant":"candidate"},
{"allocated_bytes_per_batch":702363,"allocations_per_batch":5932,"batches":500,"elapsed_ms":8569.950926,"errors":0,"gc_cycles":120,"gc_pause_ns":4889415,"go":"go1.23.2","gomaxprocs":2,"p50_ms":16.338389,"p95_ms":22.737725,"p99_ms":29.052451,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":11668.678253059112,"variant":"candidate"},
{"allocated_bytes_per_batch":1112212,"allocations_per_batch":5533,"batches":500,"elapsed_ms":9349.076452,"errors":0,"gc_cycles":189,"gc_pause_ns":7142869,"go":"go1.23.2","gomaxprocs":2,"p50_ms":17.793144,"p95_ms":24.444672,"p99_ms":30.307409,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":10696.243689247776,"variant":"baseline"},
{"allocated_bytes_per_batch":1113323,"allocations_per_batch":5534,"batches":500,"elapsed_ms":9532.681777,"errors":0,"gc_cycles":194,"gc_pause_ns":7947690,"go":"go1.23.2","gomaxprocs":2,"p50_ms":17.91347,"p95_ms":25.878835,"p99_ms":30.679071,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":10490.227444838789,"variant":"baseline"},
{"allocated_bytes_per_batch":701348,"allocations_per_batch":5931,"batches":500,"elapsed_ms":8800.89413,"errors":0,"gc_cycles":120,"gc_pause_ns":5651687,"go":"go1.23.2","gomaxprocs":2,"p50_ms":16.751363,"p95_ms":23.175394,"p99_ms":26.865442,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":11362.481870918722,"variant":"candidate"},
{"allocated_bytes_per_request":29809,"elapsed_ms":1776.706205,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":10.945647,"p50_ms":3.44581,"p95_ms":3.857471,"p99_ms":5.066898,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":281.4196284072751,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":27110,"elapsed_ms":1710.035205,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":18.280513,"p50_ms":3.239121,"p95_ms":3.760718,"p99_ms":6.65845,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":292.39164114168045,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":27125,"elapsed_ms":1729.285275,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":21.656107,"p50_ms":3.28774,"p95_ms":4.000921,"p99_ms":6.144387,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":289.1367938121141,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":29667,"elapsed_ms":1802.85585,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":16.266036,"p50_ms":3.465352,"p95_ms":3.921837,"p99_ms":6.694061,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":277.33775831273476,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":29651,"elapsed_ms":1769.872668,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":9.496904,"p50_ms":3.431853,"p95_ms":3.839032,"p99_ms":6.317466,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":282.5061989148702,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":27154,"elapsed_ms":1736.009547,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":19.043368,"p50_ms":3.286713,"p95_ms":3.989935,"p99_ms":6.998807,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":288.0168492529609,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1611637,"elapsed_ms":10928.413506,"errors":0,"gc_cycles":363,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":48.350055,"p50_ms":20.652059,"p95_ms":27.66498,"p99_ms":31.871412,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":45.75229512732898,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1172286,"elapsed_ms":10370.384091,"errors":0,"gc_cycles":249,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":39.306684,"p50_ms":19.306546,"p95_ms":26.027967,"p99_ms":31.452733,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":48.214221923942816,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1170358,"elapsed_ms":10423.401256,"errors":0,"gc_cycles":249,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":31.828335,"p50_ms":19.61991,"p95_ms":26.378625,"p99_ms":30.216222,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":47.968987062853984,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1614554,"elapsed_ms":11011.29744,"errors":0,"gc_cycles":373,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":35.865924,"p50_ms":20.822,"p95_ms":27.402202,"p99_ms":31.65621,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":45.40790971495181,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1609148,"elapsed_ms":11039.471209,"errors":0,"gc_cycles":371,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":36.610664,"p50_ms":20.783905,"p95_ms":27.866496,"p99_ms":30.189626,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":45.29202445787184,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1170029,"elapsed_ms":10480.5909,"errors":0,"gc_cycles":250,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":34.817608,"p50_ms":19.791446,"p95_ms":26.242822,"p99_ms":30.360037,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":47.70723375911945,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":29631,"elapsed_ms":1465.626077,"errors":0,"gc_cycles":5,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":23.412203,"p50_ms":11.1501,"p95_ms":14.918825,"p99_ms":20.00012,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":341.151135235976,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":27090,"elapsed_ms":1472.567985,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":23.124504,"p50_ms":11.651038,"p95_ms":13.701764,"p99_ms":16.673897,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":339.5428972333661,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":27089,"elapsed_ms":1385.685885,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":24.64915,"p50_ms":10.607873,"p95_ms":13.462288,"p99_ms":19.506038,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":360.8321376528996,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":29639,"elapsed_ms":1422.786994,"errors":0,"gc_cycles":5,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":24.012767,"p50_ms":11.047749,"p95_ms":12.915565,"p99_ms":17.411545,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":351.42294813527093,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":29629,"elapsed_ms":1432.879987,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":20.169322,"p50_ms":11.159487,"p95_ms":13.195173,"p99_ms":15.819215,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":348.9475772823394,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":27118,"elapsed_ms":1370.708169,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":25.296556,"p50_ms":10.580592,"p95_ms":12.920386,"p99_ms":17.252999,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":364.774947219272,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1643374,"elapsed_ms":10549.515312,"errors":0,"gc_cycles":508,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":109.122785,"p50_ms":82.460905,"p95_ms":92.630756,"p99_ms":102.448645,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":47.39554237446847,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1206718,"elapsed_ms":10194.814975,"errors":0,"gc_cycles":364,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":254.205358,"p50_ms":77.766275,"p95_ms":91.158747,"p99_ms":130.562619,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":49.04453893730425,"samples_per_request":200,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1205029,"elapsed_ms":9930.49146,"errors":0,"gc_cycles":365,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":94.361218,"p50_ms":77.678405,"p95_ms":88.914247,"p99_ms":93.158897,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":50.34997532740439,"samples_per_request":200,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1642924,"elapsed_ms":10695.145367,"errors":0,"gc_cycles":506,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":122.166406,"p50_ms":82.640208,"p95_ms":97.699528,"p99_ms":107.117797,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":46.750182708386184,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1645020,"elapsed_ms":10717.304647,"errors":0,"gc_cycles":505,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":111.591853,"p50_ms":83.690673,"p95_ms":96.97152,"p99_ms":105.165919,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":46.65352124145883,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1199721,"elapsed_ms":9744.524762,"errors":0,"gc_cycles":361,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":97.295218,"p50_ms":76.099123,"p95_ms":89.016934,"p99_ms":94.166268,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":51.310865559068915,"samples_per_request":200,"variant":"candidate","workers":4}
]
```

</details>

<details><summary>首轮全部 30 轮原始 JSON（含尾部回退）</summary>

head_sha: 8e9553a2ff9b48614ce4b2b79a105fce1d7ca1f4; baseline: e39e14bf8500efbd74548ef012d244eaf8f320b1; run_id: 39920705132

```json
[
{"allocated_bytes_per_batch":1112679,"allocations_per_batch":5533,"batches":500,"elapsed_ms":11656.69234,"errors":0,"gc_cycles":189,"gc_pause_ns":5221305,"go":"go1.23.2","gomaxprocs":2,"p50_ms":13.069595,"p95_ms":65.329368,"p99_ms":213.902935,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":8578.762918606806,"variant":"baseline"},
{"allocated_bytes_per_batch":703318,"allocations_per_batch":5932,"batches":500,"elapsed_ms":7023.44564,"errors":0,"gc_cycles":119,"gc_pause_ns":3882131,"go":"go1.23.2","gomaxprocs":2,"p50_ms":11.468147,"p95_ms":29.763118,"p99_ms":51.442145,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":14238.02576764871,"variant":"candidate"},
{"allocated_bytes_per_batch":702302,"allocations_per_batch":5932,"batches":500,"elapsed_ms":8839.730348,"errors":0,"gc_cycles":119,"gc_pause_ns":3785194,"go":"go1.23.2","gomaxprocs":2,"p50_ms":11.505754,"p95_ms":39.270485,"p99_ms":141.558411,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":11312.562268669782,"variant":"candidate"},
{"allocated_bytes_per_batch":1112269,"allocations_per_batch":5534,"batches":500,"elapsed_ms":7890.86539,"errors":0,"gc_cycles":191,"gc_pause_ns":5019634,"go":"go1.23.2","gomaxprocs":2,"p50_ms":12.621402,"p95_ms":34.521361,"p99_ms":65.756168,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":12672.881244017773,"variant":"baseline"},
{"allocated_bytes_per_batch":1112063,"allocations_per_batch":5534,"batches":500,"elapsed_ms":8920.219909,"errors":0,"gc_cycles":194,"gc_pause_ns":5997869,"go":"go1.23.2","gomaxprocs":2,"p50_ms":13.110254,"p95_ms":38.564939,"p99_ms":83.327298,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":11210.48595439958,"variant":"baseline"},
{"allocated_bytes_per_batch":701914,"allocations_per_batch":5932,"batches":500,"elapsed_ms":7809.696367,"errors":0,"gc_cycles":120,"gc_pause_ns":3458190,"go":"go1.23.2","gomaxprocs":2,"p50_ms":11.471666,"p95_ms":38.896233,"p99_ms":77.084967,"persisted_events":100600,"persisted_samples":200,"samples_per_batch":200,"samples_per_second":12804.59512133553,"variant":"candidate"},
{"allocated_bytes_per_request":29653,"elapsed_ms":1737.954667,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":85.975969,"p50_ms":2.437953,"p95_ms":3.691538,"p99_ms":28.199913,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":287.69450060690224,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":27130,"elapsed_ms":1356.415279,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":34.521835,"p50_ms":2.241385,"p95_ms":2.71391,"p99_ms":18.858371,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":368.618672865893,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":27131,"elapsed_ms":1517.631725,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":37.805497,"p50_ms":2.267524,"p95_ms":3.561948,"p99_ms":29.994557,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":329.4606931072161,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":29673,"elapsed_ms":1736.162822,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":69.490054,"p50_ms":2.409072,"p95_ms":3.699279,"p99_ms":47.048646,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":287.99142203956256,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":29675,"elapsed_ms":1635.547542,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":80.860417,"p50_ms":2.406906,"p95_ms":5.790826,"p99_ms":19.972449,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":305.7080195838172,"samples_per_request":1,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":27109,"elapsed_ms":1266.022984,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":33.554498,"p50_ms":2.236778,"p95_ms":2.787533,"p99_ms":9.934138,"persisted_events":503,"persisted_samples":1,"requests":500,"requests_per_second":394.937537721669,"samples_per_request":1,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1620900,"elapsed_ms":10174.139991,"errors":0,"gc_cycles":358,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":196.209399,"p50_ms":14.128122,"p95_ms":49.514833,"p99_ms":105.072919,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":49.14420289501597,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1180101,"elapsed_ms":10729.103536,"errors":0,"gc_cycles":249,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":217.697045,"p50_ms":12.877712,"p95_ms":66.342172,"p99_ms":168.879243,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":46.602215956097375,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1179325,"elapsed_ms":8305.563321,"errors":0,"gc_cycles":250,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":102.7522,"p50_ms":12.897721,"p95_ms":34.961219,"p99_ms":65.030934,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":60.20061261056034,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":1627434,"elapsed_ms":9247.583653,"errors":0,"gc_cycles":368,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":349.436329,"p50_ms":14.086557,"p95_ms":32.045157,"p99_ms":65.572481,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":54.06817810594181,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1619752,"elapsed_ms":9695.002288,"errors":0,"gc_cycles":366,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":221.796202,"p50_ms":14.104797,"p95_ms":39.855424,"p99_ms":83.287843,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":51.572963589588376,"samples_per_request":200,"variant":"baseline","workers":1},
{"allocated_bytes_per_request":1179968,"elapsed_ms":8881.027982,"errors":0,"gc_cycles":250,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":145.538696,"p50_ms":13.226983,"p95_ms":39.28924,"p99_ms":85.765422,"persisted_events":100600,"persisted_samples":200,"requests":500,"requests_per_second":56.29978883226089,"samples_per_request":200,"variant":"candidate","workers":1},
{"allocated_bytes_per_request":29613,"elapsed_ms":1274.225502,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":113.401388,"p50_ms":7.659719,"p95_ms":17.085798,"p99_ms":75.362347,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":392.3952229924841,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":27086,"elapsed_ms":1238.362053,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":84.300648,"p50_ms":7.215646,"p95_ms":26.906682,"p99_ms":61.866186,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":403.75914199625436,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":27094,"elapsed_ms":1171.865653,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":86.880316,"p50_ms":7.189491,"p95_ms":16.925214,"p99_ms":73.497558,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":426.67006983265514,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":29635,"elapsed_ms":1115.49804,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":97.265373,"p50_ms":7.682433,"p95_ms":10.999895,"p99_ms":32.042987,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":448.23028106799717,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":29645,"elapsed_ms":1115.317116,"errors":0,"gc_cycles":5,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":78.685064,"p50_ms":7.62784,"p95_ms":12.780992,"p99_ms":52.11391,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":448.3029918819968,"samples_per_request":1,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":27115,"elapsed_ms":1042.180953,"errors":0,"gc_cycles":4,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":60.233914,"p50_ms":7.144256,"p95_ms":12.616018,"p99_ms":49.124267,"persisted_events":512,"persisted_samples":4,"requests":500,"requests_per_second":479.7631338019666,"samples_per_request":1,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1656374,"elapsed_ms":8246.525126,"errors":0,"gc_cycles":512,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":183.709277,"p50_ms":56.196248,"p95_ms":115.731946,"p99_ms":158.954925,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":60.63159844424392,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1216733,"elapsed_ms":7594.22102,"errors":0,"gc_cycles":376,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":191.759744,"p50_ms":52.060744,"p95_ms":112.25491,"p99_ms":164.544878,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":65.83953754877679,"samples_per_request":200,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1221793,"elapsed_ms":7646.867535,"errors":0,"gc_cycles":378,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":178.696889,"p50_ms":52.205105,"p95_ms":124.905681,"p99_ms":160.769114,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":65.38625099905042,"samples_per_request":200,"variant":"candidate","workers":4},
{"allocated_bytes_per_request":1656539,"elapsed_ms":8109.893562,"errors":0,"gc_cycles":511,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":203.032264,"p50_ms":56.732272,"p95_ms":108.426607,"p99_ms":154.242875,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":61.65309028749988,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1653893,"elapsed_ms":7994.788609,"errors":0,"gc_cycles":510,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":176.631931,"p50_ms":56.009854,"p95_ms":112.790791,"p99_ms":143.801436,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":62.540740531542426,"samples_per_request":200,"variant":"baseline","workers":4},
{"allocated_bytes_per_request":1222831,"elapsed_ms":7509.377551,"errors":0,"gc_cycles":382,"go":"go1.23.2","gomaxprocs":2,"kind":"loopback-http-handler-real-mysql","max_ms":169.104839,"p50_ms":52.195032,"p95_ms":104.477438,"p99_ms":152.0903,"persisted_events":102400,"persisted_samples":800,"requests":500,"requests_per_second":66.58341475098912,"samples_per_request":200,"variant":"candidate","workers":4}
]
```

</details>

未合并、未发布。

<details><summary>历史实现、实验及全部原始数据</summary>

## 当前实现与验收提交

当前代码提交 **e39e14bf8500efbd74548ef012d244eaf8f320b1**。本轮继续处理上一版单线程缓存回退、SQL 形状驱逐和真实 MySQL 吞吐未改善；**此提交的真实双库 CI、必需检查和 A/B 均已通过**。

- MySQL DSN 未指定参数模式时使用驱动的 interpolateParams=true，减少逐次临时 prepare/execute/close；明确 DSN false 保持二进制 prepare。强制 utf8mb4、禁用多语句并保留驱动对不安全 collation 的拒绝与保守回退。
- JSON 写入明确使用 json.RawMessage，BLOB 保持 []byte；覆盖健康样本/事件、日统计、MFA 恢复码哈希、规则、围栏和 PG 导入。没有自行拼接用户参数。
- **取代上一版 64×8 数组缓存**：原生字符串 map 不可变热快照、不可变计划指针，免去采样散列、8 路扫描和计划结构值复制。慢路径全局 FIFO；只在稳定的 32 次慢路径窗口发布，形状抖动时不复制整表。
- 两张表各自最多 4096 项，**额外常驻内存是明确代价**。1024 形状的收益包含容量增加；8192 形状仍有回退，不宣称所有动态 SQL 都更快。
- 保留前几轮顺序参数零复制、HTTP 数值直方图、uint8 批次索引和完整 SQL 模板；业务事务、版本历史、归属、通知、持久性设置保持。
- 新增参数模式/特殊字符/JSON/非 UTF-8 BLOB/时间/NULL/NO_BACKSLASH_ESCAPES/prepare 计数与安全回退回归。随机等价及既有双库用例继续运行。

## 本轮局部对照与验证

同一宿主、同一基准文件、Go 1.23.12，各三轮 300 ms，中位数，基线 **8a3da7c**：

| 缓存路径 | 上一版 ns/op | 当前 ns/op |
| --- | ---: | ---: |
| 1P、共享长模板 | 48.25 | 7.759 |
| 8P、共享长模板 | 5.689 | 1.485 |
| 1P、14 参数 | 30.63 | 15.6 |
| 8P、14 参数 | 4.986 | 2.367 |

并发 ns/op 是吞吐摊销成本，不是单请求延迟。[实现、所有缓存档位及原始结果](https://github.com/WatchTestCs/smart-elderly-watch-management/tree/perf/backend-hot-path-20261023/docs/performance/2026-10-23-transport)。

本地最终短模式 race/build/vet/gofmt 通过；数据库用例本地跳过。最终提交 Go 1.23.2/1.27.2 的真实 MySQL/PG race 测试、构建、vet、格式检查通过；Preview CI required、前端、依赖审查、CodeQL Go/JS 与新版 A/B 均已通过。A/B 已将基线更新为 8a3da7c，依然同 runner、交替各三次，每轮 500×200 条实际 SaveBatch+commit，核对 100,600 条历史事件。最终实测见下表，不能仅依赖缓存纳秒数据。

### 已完成的真实 MySQL 对照

[a6a52f0 的 A/B 运行](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39917496292)，相对 8a3da7c 三轮中位数：吞吐 **9773.3→10812.4 样本/s（+10.6%）**；p50 **17.322→15.519 ms（-10.4%）**；p95 **35.366→33.342 ms（-5.7%）**；p99 **59.946→58.378 ms（-2.6%）**。后两轮候选 p99 高于对应基线，不能承诺稳定尾延迟改善。全部零错误且持久化数量正确。

代价：**915034→1111083 B/批（+21.4%）**，**GC 155→195（+25.8%）**。更大编码缓冲换取该场景的传输吞吐；不代表所有业务或硬件饱和。

a6a52f0 首次双库 CI 发现日统计和 MFA 三处 JSON 字节参数遗漏（MySQL 3144），已在 **e39e14bf8500efbd74548ef012d244eaf8f320b1** 明确 RawMessage，原业务断言保持。最终 SHA 的双库 CI 和 A/B 已重跑通过；修复前结果作为历史保留。

<details><summary>a6a52f0 的六轮全部原始结果</summary>

```json
{
  "run_id": 39917496292,
  "head_sha": "a6a52f054e89f3c4051763c65cbf387b8db6ee67",
  "baseline_sha": "8a3da7c3795eff02e67beb0af0a87a95c3156704",
  "results": [
    {
      "allocated_bytes_per_batch": 914244,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 11205.402992,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 6495855,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 17.321528,
      "p95_ms": 43.971131,
      "p99_ms": 76.81519,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 8924.266273278536,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 1111083,
      "allocations_per_batch": 5533,
      "batches": 500,
      "elapsed_ms": 9386.744314,
      "errors": 0,
      "gc_cycles": 193,
      "gc_pause_ns": 7666617,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 15.519113,
      "p95_ms": 33.672796,
      "p99_ms": 58.168134,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10653.320965699844,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 1110997,
      "allocations_per_batch": 5533,
      "batches": 500,
      "elapsed_ms": 9176.544891,
      "errors": 0,
      "gc_cycles": 195,
      "gc_pause_ns": 8803643,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 15.373521,
      "p95_ms": 33.341624,
      "p99_ms": 58.378138,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10897.347660564066,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 915034,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 10232.006702,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 6505186,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 17.321543,
      "p95_ms": 35.154005,
      "p99_ms": 54.820767,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 9773.253958136433,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 915154,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 9841.292694,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 6226093,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 16.649973,
      "p95_ms": 35.366422,
      "p99_ms": 59.945944,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10161.266726775397,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 1111396,
      "allocations_per_batch": 5533,
      "batches": 500,
      "elapsed_ms": 9248.623226,
      "errors": 0,
      "gc_cycles": 196,
      "gc_pause_ns": 7496002,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 15.615524,
      "p95_ms": 31.739488,
      "p99_ms": 59.702381,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10812.42013609951,
      "variant": "candidate"
    }
  ]
}
```

</details>

### 最终提交 e39e14b 的真实 MySQL A/B

[运行 39918229905](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39918229905)，相同方法重新测最终提交，每轮均零错误、200 条最终样本和 100,600 条历史事件。三轮对应吞吐均增加。

| 指标（三轮中位数） | 基线 8a3da7c | 最终 e39e14b | 变化 |
| --- | ---: | ---: | ---: |
| 样本/s | 10376.7 | 11085.6 | +6.8% |
| 批事务 p50 ms | 18.363 | 17.150 | -6.6% |
| 批事务 p95 ms | 24.550 | 24.259 | -1.2% |
| 批事务 p99 ms | 31.223 | 32.593 | +4.4% |
| 分配 B/批 | 915154 | 1112842 | +21.6% |
| GC 次数/轮 | 152 | 192 | +26.3% |

接受吞吐与内存/尾延迟的明确交换：**本次真实吞吐提高，但 p99 变差**。不能将此前 a6a52f0 的 +10.6% 直接当作最终提交数字，也不能承诺硬件已饱和或生产 SLA 改善。

<details><summary>最终提交六轮全部结构化原始结果</summary>

```json
{
  "run_id": 39918229905,
  "head_sha": "e39e14bf8500efbd74548ef012d244eaf8f320b1",
  "baseline_sha": "8a3da7c3795eff02e67beb0af0a87a95c3156704",
  "results": [
    {
      "allocated_bytes_per_batch": 914723,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 10234.817522,
      "errors": 0,
      "gc_cycles": 152,
      "gc_pause_ns": 4873734,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 19.304881,
      "p95_ms": 26.600186,
      "p99_ms": 31.222795,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 9770.569898783975,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 1112320,
      "allocations_per_batch": 5533,
      "batches": 500,
      "elapsed_ms": 8880.912346,
      "errors": 0,
      "gc_cycles": 189,
      "gc_pause_ns": 6187284,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 16.704474,
      "p95_ms": 24.258944,
      "p99_ms": 32.593433,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 11260.104379370483,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 1112842,
      "allocations_per_batch": 5534,
      "batches": 500,
      "elapsed_ms": 9328.865861,
      "errors": 0,
      "gc_cycles": 192,
      "gc_pause_ns": 7892615,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 17.487942,
      "p95_ms": 26.166121,
      "p99_ms": 33.826627,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10719.416646138867,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 915154,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 9602.482477,
      "errors": 0,
      "gc_cycles": 152,
      "gc_pause_ns": 5412136,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 18.359233,
      "p95_ms": 24.550293,
      "p99_ms": 28.205655,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10413.973703104524,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 915550,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 9636.93973,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 5120692,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 18.362595,
      "p95_ms": 23.783978,
      "p99_ms": 34.966592,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10376.73813489752,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 1113852,
      "allocations_per_batch": 5534,
      "batches": 500,
      "elapsed_ms": 9020.690097,
      "errors": 0,
      "gc_cycles": 192,
      "gc_pause_ns": 7329803,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 17.150369,
      "p95_ms": 23.952618,
      "p99_ms": 30.019741,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 11085.626368348125,
      "variant": "candidate"
    }
  ]
}
```

</details>

最终验收：[双库 CI 39918237795](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39918237795)、[安全检查 39918237761](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39918237761)、[MySQL A/B 39918229905](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39918229905)，均对应 **e39e14bf8500efbd74548ef012d244eaf8f320b1**。

未合并、未发布；独立于 #27/#29。

<details>
<summary>保留上一版实验：分配和 GC 改善，但真实 MySQL 吞吐中位数下降 2.7%</summary>

## 行为与实现

在 preview 上优化 SQL 参数绑定、缓存读路径、HTTP 指标和健康批次内存布局，保留业务事务、设备归属、版本历史、通知、MFA 撤销语义及发布版本。

- 顺序 SQL 参数直接复用只读切片；重复/乱序参数仍重排，缺参仍报错。
- SQL 计划改为 **64 组×8 路不可变数组快照**，命中仅原子读；未命中按组加锁、编译和发布，最多 512 个已发布条目。长 SQL 抽样仅用于路由，始终用完整字符串验证。顺序计划不再常驻恒等位置向量。
- HTTP 指标使用连续数值桶；指标名称只初始化一次；抓取在数值快照后处理，保留累计桶与慢抓取隔离。
- 健康批次只存一份归一化样本，最终映射与事件列表使用 **uint8 索引**；200 条边界双重校验。保留全部中间版本、冲突和删除标记。
- 缓存三类健康批次的完整 SQL，避免每批重新拼接已缓存的 VALUES 片段。
- 新增随机等价、碰撞/驱逐/快照、竞态、真实 MySQL 回归及同 runner 交替 A/B 工作流；同步架构报告和文件索引。

## 本轮相对第一版 8f530cc 的局部对照

Go 1.23.12，AMD EPYC 9V74，8 核 cgroup。中位数；原始值与方法见 `docs/performance/2026-10-23-hardware/`。

| 路径 | 原方案 | 新方案 |
| --- | ---: | ---: |
| 8P、14 参数热缓存（并发摊销 ns/op） | 49.64 | 4.349 |
| 8P、64 条 SQL 工作集（并发摊销 ns/op） | 44.46 | 5.786 |
| 200 身份的版本选择 | 58.87 μs；152,575 B；209 次分配 | 25.04 μs；27,228 B；12 次分配 |
| 批长 200 的三条 SQL 构造/获取 | 4.683 μs；24,320 B | 6.191 ns；0 B |

**明确代价：** 单线程缓存命中变慢（共享长模板 8.957→44.94 ns，14 参数 16.04→23 ns）；1024 SQL 超容量、4P 时 3.190→4.853 μs。分组 FIFO 可能提前驱逐，未命中会复制快照，模板占用常驻内存。该实现选择稳定工作集的多核吞吐，不是所有输入都更快。并发 ns/op 不是单次请求延迟，不能相加换算成整个服务的倍数或 p99。

前一轮相对 preview@3dbd79d 的 HTTP 中间件局部对照仍保留在 `docs/performance/2026-10-23-hot-path/`。

## 验证与数据库实验

- 本地最终 `go test -race -short ./...`、`go build ./...`、`go vet ./...` 通过；无本地 MySQL/PG，相关用例跳过。
- 新测试对照原值复制算法的 1000 组随机输入；验证碰撞不会混淆 SQL、快照不变、缺参重排、索引溢出，以及 MySQL 中的 200 条历史、50 条最终版本与重复重放。
- **当前代码提交：8a3da7c3795eff02e67beb0af0a87a95c3156704。** 先前 8f530cc 双库 Go 1.23.2/1.27.2 CI 全绿不能代替此提交的验收；本提交 Go 1.23.2/1.27.2 的真实双库 race/build/vet/gofmt、Preview CI required、前端、CodeQL 和硬件 A/B 已全部通过。
- 新工作流在同 runner 的专用 MySQL 8.4、Go 1.23.2、GOMAXPROCS=2 上交替执行 baseline/candidate 各三次；每次独立合成库，预热 3 批再测 500×200 条真实 SaveBatch+commit，并验证 100,600 条历史事件。首轮 100 批原始数据保存在 mysql-initial.json：每批分配中位数减少约 14.2%、GC 次数减少约 18.9%，但吞吐没有改善，候选一次 p99 达 156.7 ms；未删除该轮或归因为外部因素。已扩为 500 批继续观察。串行短测不代表入口排队或生产 SLA。

独立于 #27/#29；未合并、未发布二进制。

## 真实 MySQL 500 批对照（所有轮次）

[运行 39914348329](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/39914348329)，同 runner、MySQL 8.4、Go 1.23.2、GOMAXPROCS=2；每轮预热 3 批，再串行测 500×200 条，baseline/candidate 交替顺序。每轮均零错误、200 条最终样本和 100,600 条历史事件。

| 轮次 | 基线 样本/s | 候选 样本/s | 基线 p99 ms | 候选 p99 ms |
| --- | ---: | ---: | ---: | ---: |
| 1 | 8973.9 | 9872.5 | 62.239 | 29.447 |
| 2 | 10149.6 | 9741.2 | 28.863 | 30.887 |
| 3 | 10214.5 | 10086.7 | 28.812 | 28.064 |

每批分配中位数 **1,067,057→914,871 B（-14.3%）**，GC 次数 **188→155（-17.6%）**；吞吐中位数 **10,149.6→9,872.5 样本/s（-2.7%）**。批事务 p99 中位数 28.863→29.447 ms，没有证据证明此串行 MySQL 场景整体加速。保留该结果；不能用局部 CPU 纳秒基准替代实际数据库吞吐。

本轮收益是减少分配、GC 和稳定 SQL 工作集的多核共享争用；单线程、容量抖动及数据库事务墙钟成本明确存在代价。500 批仍是短期串行实验，未测生产队列/混合业务/长期稳定性；第一轮 100 批中的候选长尾同样保留在仓库。

<details>
<summary>六轮结构化原始结果（含所有成功轮次）</summary>

```json
{
  "run_id": 39914348329,
  "head_sha": "8a3da7c3795eff02e67beb0af0a87a95c3156704",
  "results": [
    {
      "allocated_bytes_per_batch": 1066632,
      "allocations_per_batch": 5745,
      "batches": 500,
      "elapsed_ms": 11143.419117,
      "errors": 0,
      "gc_cycles": 188,
      "gc_pause_ns": 9761292,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 20.210604,
      "p95_ms": 28.059592,
      "p99_ms": 62.239448,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 8973.906388160845,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 915584,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 10129.127377,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 7140974,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 19.205436,
      "p95_ms": 26.172024,
      "p99_ms": 29.446921,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 9872.518754879906,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 914801,
      "allocations_per_batch": 5546,
      "batches": 500,
      "elapsed_ms": 10265.697285,
      "errors": 0,
      "gc_cycles": 155,
      "gc_pause_ns": 6939601,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 19.352841,
      "p95_ms": 26.168609,
      "p99_ms": 30.886587,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 9741.179505275077,
      "variant": "candidate"
    },
    {
      "allocated_bytes_per_batch": 1067057,
      "allocations_per_batch": 5745,
      "batches": 500,
      "elapsed_ms": 9852.608672,
      "errors": 0,
      "gc_cycles": 186,
      "gc_pause_ns": 9489386,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 18.735006,
      "p95_ms": 25.720115,
      "p99_ms": 28.862577,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10149.59624695018,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 1067457,
      "allocations_per_batch": 5745,
      "batches": 500,
      "elapsed_ms": 9790.047383,
      "errors": 0,
      "gc_cycles": 188,
      "gc_pause_ns": 8861832,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 18.671862,
      "p95_ms": 25.303058,
      "p99_ms": 28.811994,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10214.455159190113,
      "variant": "baseline"
    },
    {
      "allocated_bytes_per_batch": 914871,
      "allocations_per_batch": 5545,
      "batches": 500,
      "elapsed_ms": 9914.044585,
      "errors": 0,
      "gc_cycles": 153,
      "gc_pause_ns": 6428155,
      "go": "go1.23.2",
      "gomaxprocs": 2,
      "p50_ms": 18.872328,
      "p95_ms": 25.799361,
      "p99_ms": 28.06434,
      "persisted_events": 100600,
      "persisted_samples": 200,
      "samples_per_batch": 200,
      "samples_per_second": 10086.700654070139,
      "variant": "candidate"
    }
  ]
}
```
</details>


</details>


</details>

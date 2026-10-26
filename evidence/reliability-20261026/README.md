# 证据索引与使用边界

每个结果绑定其 manifest/summary/source-sha。测试计数含子测试，不是独立业务用例数。
真实本机数据库与真实 Linux CI 分开记录；本机进程接收端不是外部人类值班渠道。

| 目录/文件 | 绑定与解释 |
|---|---|
| remote-baseline.json | 初始远端 preview 41b0f1d、PR 元数据。历史交接的“无当前 CI”不再适用于该新 preview：其三个 exact-head workflow 已成功 |
| baseline-50b9cc0、baseline-41b0f1d | 最初 HDD 跑出的真实失败，DDL/迁移超时；保留失败，不能当通过 |
| baseline-50b9cc0-nvme | 50b9cc0 真 MySQL+PG/race：1041 pass、10 opt-in skip，build/vet 成功 |
| baseline-41b0f1d-nvme | 41b0f1d：1068 pass、10 opt-in skip，build/vet 成功 |
| candidate-acceptance-nvme | 2d0f308 的较早候选：1095 pass、10 skip；不替代后续代码的 CI |
| ci-bf33177-failed | 新增 rollback 注释分割、迁移计数、Linux 客户端 localhost socket 三项失败；原始 artifact 和 job log 保留 |
| ci-refinement-rollback-restore.txt | 定向恢复/迁移修复验证；其中 db 测试因本地 DSN 指向不存在的基库失败，不记通过 |
| rollback-refinement-correct-db.txt | 使用实际 care_lab 基库重跑 rollback/reapply 成功 |
| ci-f1417a9-backend-* | SHA f1417a9，两个 Go 版本，真实 MySQL 8.4.13/PG 16.15/race；每个主套件 1106 pass/10 opt-in skip；PG 专项 825 pass/42 解释过的 skip（MySQL-only 在配对套件必跑） |
| ci-f1417a9-frontend | 同 SHA 的 test/typecheck/build/npm audit 原始记录；21 tests，通过；audit 0 |
| verified-source-f1417a9.json | 实际在线运行 release-gate.py 校验此新分支的当时 head、最新 CI/Security 与 SHA256 artifact；不是永不过期发布许可 |
| process-smoke-1 | 2d0f308 之后的未提交候选 setup smoke；旧 manifest 只有基准 SHA，**代码是 dirty**；complete=false，不是验收 |
| process-matrix-bf33177 | 干净 bf33177 的首轮真实进程演练，部分通过后脚本等待失败；heartbeat 场景可能收到旧 firing，不能据此验收；complete=false |
| process-matrix-110b5a4 | 修正告警归属与等待后，从干净 110b5a4 构建/运行；以 matrix.json.complete 和逐项记录为准，真实租约和升级 deadline |
| process-matrix-110b5a4/final-audit-timeline.json | 演练后只读恢复同一测试实例检查 audit；ack 后 614.111 秒真实升级，无任务状态改写；之后停掉独立实验数据库服务，保留数据 |
| *dirty*.txt、blob-integration-*.txt | 未提交候选的开发/微调记录，含真实失败与修复；不冒充最终 immutable 验收 |
| http-audit-*、http-tail-* | 历史基线/候选的本机 HTTP 逐请求 latency；overlay 仅延长冷路径样本，详见 docs/reliability/performance.md |
| limiter-memory-bf33177 | 模拟来源微压测和 Windows RSS 采样，不是 HTTP 吞吐 |
| promtool-*.txt | 正式规则检查/单测。Windows 单测只把 Docker 绝对路径改为本地规则绝对路径，未缩短 for；初次路径失败不用于验收 |
| schema-bf33177.tsv | 只含 information_schema 元数据，不含业务行；385 字段分类输入 |
| python-gates-and-watchdog.txt | 5 个发布失败关闭测试 + 2 个 watchdog 持久重试/顺序与接收端边界测试 |

原始凭据、TLS 私钥、fixture session、SQL dump 和 S3 数据均未放入此目录。截图不能代替原始
日志。运行日志中的测试姓名/号码来自合成 fixture。后续文档/证据提交产生新 SHA 后必须重新
跑 exact-source CI；本目录不拿旧绿灯替代新提交。

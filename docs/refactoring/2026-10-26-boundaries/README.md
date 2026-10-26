# 请求边界与会话生命周期重构 · 2026-10-26

基线：`preview@50b9cc094002ca691244b7d504cdeedf117787dd`。本轮处理可复现的资源生命周期问题、请求取消脱节和边界规则重复；没有以代码行数、文件数量或新增抽象当作质量提升证据。

## 已处理的问题

| 坏味道 | 修改及保留的合同 |
| --- | --- |
| 会话存储创建即启动永不退出的清理任务 | 新增幂等并发 Close，取消正在执行的清理 SQL 并等待任务退出；服务在 SQL 池关闭前清理 Store。Close 不关闭外部 SQL 池、不撤销有效会话 |
| 会话读写脱离请求使用 Background | HTTP 调用传递请求 context，数据库操作保留 3 秒上限；已取消请求不复用已验证快照。原 Get/Create/Destroy/Revoke 包装保留给无请求的调用者 |
| 未区分正常请求与提交后的必要清理 | 审计失败后的登录补偿撤销、MFA 已提交后的内存撤销显式使用 WithoutCancel；存储 SQL 仍受 3 秒上限约束，不自动重放写入 |
| 模型、加密 SQL、Cookie/请求上下文与清理循环混在一个文件 | session.go 管模型、内存和后端选择；database.go 管加密 SQL 与代际校验；http.go 管 Cookie/请求快照；cleanup.go 管周期清理和失败日志；使用具体私有后端，未添加通用 Repository 框架 |
| 请求 JSON 解码重复三份且清理方式不同 | request_json.go 统一大小限制、关闭和单文档检查；显式保留普通/可选请求的 1 MiB、健康/能力请求的 2 MiB，以及能力端点拒绝未知字段等差异 |
| auth、CSRF 和 OpenAPI 各自保存权限表 | accesspolicy 一份类型化规则，由两个中间件与文档生成脚本读取。39 条显式规则逐项保持原样：6 公开、10 设备、23 共享。未知业务路由仍走管理员权限；非业务路由拒绝、静态放行和路径级 CSRF 语义继续保留 |
| 重复拒绝响应、令牌生成忽略随机源错误、清理错误静默 | 合并相同拒绝响应；令牌生成失败拒绝创建；会话创建时间和到期时间来自同一时钟值；清理失败记录不含凭据/标识的结构化事件，并继续下一轮 |
| 测试夹具隐式丢弃 Store 所有权 | testHandler 接收 testing.TB 并注册 Close；直接构造的 Store 同样由测试/基准负责关闭。保留原业务断言与数据库清理顺序 |

“统一”不表示所有端点采用同一种严格度。JSON 错误文案、未知字段策略、空正文、null、浮点/整数类型、字节上限与尾随数据的行为均按旧版本对照。2 MiB 解析函数现在显式关闭请求体；旧版真实 HTTP 服务也会在请求结束时关闭正文，不能将直接函数调用中的未关闭现象夸大为所有网络请求都会泄漏。

## 实际复现与回归

| 观察 | 固定基线 | 本轮 |
| --- | --- | --- |
| 创建 25 个 Store，再释放支持关闭的实例 | 无 Close；保留 25 个维护协程 | Close 后保留 0 个 |
| 请求在会话写入前已取消：直接调用登出或会话轮换 | 仍改变会话、发送 Cookie 并报告成功，两个回归按预期失败 | 返回失败状态、不发送 Cookie、原会话保留，回归通过 |
| JSON 接受结果、已解析值与完整错误响应 | 原版 oracle 通过 | 4,080 组新旧比较全部一致 |
| 请求体所有权：成功、解析失败、额外文档 | 原 2 MiB 直接解析函数不主动关闭，回归按预期失败 | 三类解析入口均执行关闭 |
| 取消正在执行的会话 SQL | 原请求读取使用独立 Background | Get/Create/Destroy/Revoke/FromRequest 五条路径收到调用者取消，并释放 SQL 连接 |
| 清理与关闭 | 无退出入口；SQL 清理错误被忽略 | 活跃清理 SQL 可取消；并发 Close 可重入；失败后仍处理下一次 tick；外部数据库仍可使用 |

生命周期夹具读取 goroutine profile 中由 NewStore 创建的任务数；没有把进程其他协程算成会话任务。两版运行相同夹具，原始输出见 [基线](evidence/baseline-lifecycle.txt) 与 [本轮](evidence/candidate-lifecycle.txt)。这是小规模生命周期验证，不是长期 RSS 或生产吞吐测量。

取消写入的旧版失败和本轮通过见 [基线](evidence/baseline-cancelled-writes.txt) 与 [本轮](evidence/candidate-cancelled-writes.txt)。失败记录只输出 Cookie 是否存在，不保留会话令牌。JSON 旧合同和正文关闭差异分别见 [兼容性](evidence/baseline-json-contract.txt) 与 [所有权](evidence/baseline-body-ownership.txt)。

权限测试冻结旧版规则，覆盖全部 39 条显式规则及未知/大小写/尾斜杠路径、七种 HTTP 方法、五种身份和有无 CSRF 令牌，共 4,515 次中间件请求。另独立核对完整新旧规则键与权限相等。文档生成不再硬编码公开/设备名单；对无法识别或重复的规则字面量明确报错，不静默猜测。当前 123 条 OpenAPI 路径生成结果逐字节不变。

## 验证与限制

- `go test -race -short -json ./... -count=1 -timeout 180s`：**918 项测试/子测试通过，125 项跳过，0 失败**。
- `go build ./...`、`go vet ./...`、Go 格式及 `python3 scripts/generate-openapi.py --check` 通过。命令输出及逐项跳过原因见 [validation.txt](evidence/validation.txt)；生产源码散列见 [source-hashes.json](evidence/source-hashes.json)。
- 本机未配置 MySQL/PG 测试连接。阻塞 SQL/取消使用实际 database/sql 池加受控 driver 验证；真实加密会话、跨实例撤销、MFA、数据库故障与其他业务集成以本次提交的远端双版本 CI 为准，不把本地跳过算成通过。
- 本轮没有前端行为、数据库迁移、部署拓扑或 Windows 原生壳修改，因此没有重复不相关的前端视觉/安装检查；没有宣称生产 p99、CPU 吞吐或完整商业化验收得到提升。

保留原有 nil/false 的会话失败接口以避免扩大调用面，没有进行全库错误类型迁移。普通请求取消不能证明此前已提交的 SQL 被回滚；必要补偿撤销也仍可能因数据库故障失败，返回和审计不能承诺未验证的成功。本轮没有合并其他 PR；Cookie 实现现位于 session/http.go，后续待合并改动需按实际位置与上述合同重新核对。

## 复现与后续边界

生命周期对照只需将 `internal/session/lifecycle_audit_test.go` 原样复制到固定基线，执行 `go test ./internal/session -run '^TestStoreCleanupLifecycleAudit$' -v -count=1`。JSON 和取消写入夹具同样可复制到基线；正文所有权及取消写入在旧版预期失败，本轮应通过。不要复制本轮生产实现到基线。

当前仍保留旧接口/双数据库兼容、部分大处理器与领域规则。后续拆分应继续以明确职责、可复现问题和行为对照为依据，尤其不能因文件较长就改变事务边界、写入确认或设备协议。这里没有宣称仓库所有坏味道已经清零。

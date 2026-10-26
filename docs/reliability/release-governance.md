# 仓库和发布保护

本次只推送新分支 `codex/reliability-validation-20261026`。没有向 preview/main 合并，没有修改
原 PR 分支、保护规则、tag 或 GitHub Release。`deploy/preview-ruleset.proposed.json` 是可审查的
管理员提案，**未应用**。需独立维护者审批后由管理员导入，验证禁止普通直推、force push、删除，
至少一名独立 reviewer、最后推送者不能自行批准、新推送使旧批准失效，以及 required checks。

提案字段依据 [GitHub rules REST 文档](https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset)。
`integration_id=15368` 来自本仓库 bf331775 的实际 check-runs（slug=github-actions），不是假设。
部署时再次核对 check 名称与来源；CodeQL 的 fork PR 权限限制不能用跳过代替完成，应由维护者
在可信分支重新运行。无独立 reviewer 时不伪造批准。

## 实现的发布门槛

Preview CI 覆盖当前 head、真实 MySQL/PG、race、迁移及 rollback、S3 与真实恢复、build/vet、
OpenAPI、前端 test/typecheck/build/audit、Windows sidecar、govulncheck；Release gate 聚合所有
结果，任何失败/取消/缺项均不通过。数据库 skip 有单独审核，不能用缺少 DSN 变绿。
Security 工作流执行 CodeQL；所有 codex 分支 push 与 preview/PR 都触发检查。

release-gate.py 校验明确 40 位 SHA 等于当前 preview，两次读取防止验证期间分支移动；选择该
SHA 最新运行，拒绝重跑失败后的旧绿灯；检查聚合 job 与每个 SHA 命名 artifact 的存在、有效期、
SHA-256 digest。Release preparation 和真正 Windows installer 手动入口都要求此门槛；installer
固定 checkout SHA，产物上传前再次检查。没有证据不进入打包。

这段逻辑仍须搭配管理员保护：拥有写权限的人在无保护分支可以改工作流或另开发布入口，不能
声称 Python 脚本本身能约束仓库管理员。现存历史发布链也需发布者按此路径统一管理。

## 来源和依赖

CI artifact 保存 source SHA、Go 版本与平台、原始测试 JSONL、逐项 skip、构建/vet、前端 audit
和原生 sidecar SHA-256。GitHub artifact digest 是 ZIP 证据完整性，sidecar 文件 hash 是二进制
完整性，两者用途不同。正式安装包还需 package-release.ps1 的清单、发布 SHA 和构建 run 链接。
SDK 版本与校验和固定在 go.mod/go.sum；S3 测试服务固定镜像 digest。MySQL/PG 的 CI 标签属于
持续获得补丁的环境，实际版本从测试日志/镜像元数据读取，不能把本地 8.0.43 混称 CI 8.4。

本次不创建签名密钥、不声称 unsigned 提交已签名。生产还需维护者配置受托签名 tag、代码签名
证书、时间戳服务、证书轮换、构建证明和留存策略。所有 action 尚未全部改为 immutable SHA
（如 setup-go@v6），后续供应链加固应使用受审核 SHA 更新并保留验证记录。
仓库 default branch 的 Dependabot 历史告警不等于本新分支当前依赖扫描结果，不能用本次 CI
把默认分支旧漏洞声明为清零。

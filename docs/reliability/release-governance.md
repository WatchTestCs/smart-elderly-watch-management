# 仓库和发布保护

本指南说明仓库保护与预发布交付应满足的控制。分支保护须由仓库管理员在 GitHub 中配置；本文不代表保护已经启用。
正式启用前由仓库管理员核对目标分支、检查名称和来源，并确认规则实际生效。

建议保护规则至少包括：禁止未经审查的直接推送、force push 和分支删除；要求独立 reviewer；
最后推送者不能自行批准；新提交使旧批准失效；合并前必须通过当前 SHA 对应的必需检查。
字段说明见 [GitHub rules REST 文档](https://docs.github.com/en/rest/repos/rules#create-a-repository-ruleset)。
GitHub Actions 检查来源与名称应以仓库实际 check-runs 为准，不应硬编码未经复核的 integration ID。
CodeQL 或依赖审查因 fork 权限、配置缺失而未运行时，应由维护者在可信分支重新执行，不能把跳过当成通过。

## 预览交付

预发布只面向 `preview`，不自动更新 `main`。附件、源码快照和校验清单必须对应同一版本与提交；旧 Release 不包含后续分支改动。普通 PR/CI 不读取发布私钥，签名与发布只在明确的 release 流程执行。

## 分支通道与 main 晋升

`preview` 是预发布集成通道，允许经过评审的功能、架构、依赖与破坏性变更；`main` 是稳定通道，不接收日常开发 PR。正式晋升必须绑定一个明确的 40 位 `preview` HEAD SHA，并在候选冻结后只接受缺陷、安全、构建或发布阻断修复。

晋升前先在候选 SHA 上运行完整 Preview CI、Security 与候选打包，确认 release gate 证据完整。晋升时不得用 squash 或 rebase 生成未经候选验证的新源码提交；稳定主线应直接采用已经验证的候选 SHA。首次从历史 `main` 切换到新架构前，必须先保存旧 `main` 的不可变标签并记录其 SHA。由于当前两条历史已分叉，首次切换属于一次性世代迁移；完成后要求 `main` 始终位于后续 `preview` 历史上，常规晋升只允许 fast-forward。

候选 SHA 提升到 `main` 后，必须在 `main` 上重新运行同一套分支敏感 CI 与 Security，并通过 Stable release candidate 的 exact-source 检查后才能创建正式版本标签。若 `main` 出现独立提交或两条分支失去祖先关系，应停止常规晋升，先显式解决历史关系，不能用强制合并掩盖分叉。

详细步骤见 [main 晋升手册](main-promotion.md)。

## 当前发布门槛

`Core CI / required` 是日常合并的基础检查，不表示已经通过正式发布验收。
UI Quality 由 Preview CI 在每个候选提交上调用，不再按前端路径过滤；复用工作流明确
检出 PR head SHA，UI 日志与图片归入 `ui-quality-<SHA>`，不自动更新视觉基线。

正式候选需在当前 `preview` SHA 上手动运行 Preview CI，启用 `require_native` 与
`build_desktop`。原生构建读取显式 SDK URL / SHA-256，或仓库变量 `HARMONY_SDK_URL` /
`HARMONY_SDK_SHA256`。缺少配置必须失败；普通 PR 的可选原生 job 被跳过时，只能声明
基础检查通过。SDK 工具链版本为 Command Line Tools 6.1.1.280 / HarmonyOS API 24，
URL 必须为 HTTPS，并在解压和执行前核对预期归档 SHA-256。

`scripts/reliability/release-gate.py` 对正式候选独立检查 UI、Go 1.27.3、手表协议与真实
后端联调、原生 HAP、模拟器协议、Windows sidecar、实际 Tauri 构建和安装/卸载测试、
部署合同及漏洞扫描。另查两项 CodeQL 结果。每项必需 job 必须在最新运行 attempt 中
明确完成且成功，不能由聚合 job 的绿色状态代替，也不接受 skipped/cancelled。
源码相同但来自其他分支或 PR 事件的运行不用于正式发布认证。

必需 artifact 包括 `ui-quality-`、`watch-native-`、`watch-integration-`、
`windows-desktop-` 等 SHA 命名前缀。运行、job 与 artifact 均遍历分页；缺失、重复、
过期、空归档、摘要不合法或来源不一致都会阻断。普通 CI 绿色但没有完整软件证据时，
`preview-candidate` 仍会在打包前被拒绝。这是阻断状态，不是发布已通过。

Preview CI 与 Security 工作流分别验证当前提交的后端、数据库、race、迁移与恢复、S3、
OpenAPI、前端 test/typecheck/build/audit、Windows sidecar 及漏洞扫描。具体执行范围以当前
workflow 文件和对应 commit 的 job 结果为准；未配置的数据库测试、跳过的原生 SDK 编译、
未进行的真实设备或生产环境演练不得报告为通过。

发布门禁应校验明确的 40 位 commit SHA 与目标分支 head 一致，防止验证期间分支移动；选择
该 SHA 对应的最新工作流运行，拒绝用旧的成功运行覆盖新提交的失败/取消结果；同时核对聚合
job、SHA 命名 artifact、有效期和 SHA-256 digest。Release preparation 与 Windows installer
入口必须固定 checkout 到已验证 SHA，产物上传前再次核对来源。没有完整证据不得进入打包。

这些自动化不能代替管理员保护：有写权限的人在未保护分支仍可能修改工作流或另开发布入口。
仓库管理员必须独立启用分支规则、限制发布权限，并定期审查工作流来源和配置。

## 来源和依赖

CI artifact 应保存 source SHA、Go 版本与平台、原始测试日志、逐项 skip、构建/vet、前端 audit
和原生 sidecar SHA-256。GitHub artifact digest 是归档完整性证据，sidecar 文件 hash 是二进制
完整性证据，两者用途不同。正式安装包还需发布清单、发布 SHA 和构建 run 链接。

签名密钥不进入仓库；未签名提交或附件不得描述为已签名。生产发布还需维护者配置受托签名
tag、代码签名证书、时间戳服务、证书轮换、构建证明和留存策略。CI action、容器基础镜像、
SDK 与依赖来源应固定版本或摘要，并按补丁策略持续审核。

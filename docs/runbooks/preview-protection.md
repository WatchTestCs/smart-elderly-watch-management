# Preview 分支保护与 CI 合并门槛（P0）

> 本文件是管理设置的**执行清单**，不是已经生效的保护规则。GitHub 集成目前对 `GET /branches/preview/protection` 返回 403，无法代替有权限的仓库管理员在 GitHub UI 配置或读取实际状态。不得把本文档存在视为启用保护。

## 0. 作用域与依赖顺序

- 仅作用于 `refs/heads/preview`，**不要修改 `main` 的既有规则**；草稿 PR、依赖链必须按先祖顺序处理。
- 目前 [PR #1](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/1) 是 10 月 15 日修复整合候选，尚未合入。
- [PR #2](https://github.com/WatchTestCs/smart-elderly-watch-management/pull/2) 的 base 是 PR #1 的开发分支。**只有 PR #1 审核合格合入 preview 后**，才可以将 PR #2 的 base 改成 preview 并重新跑完整门禁。
- 合并前确认 `preview` 与当前 PR 祖先链未发生未审查的变动；不要用 force push 覆盖工作。

## 1. 依赖图与安全审查（先执行）

1. 有权限的管理员打开 [仓库 Security & analysis](https://github.com/WatchTestCs/smart-elderly-watch-management/settings/security_analysis)。
2. 启用仓库支持的 Dependency Graph（如选项不可用，明确记录账户/仓库能力限制；不视作已通过）。
3. 重新运行对应 PR 的 **Security → Dependency review**，并分别确认 Java 和 JavaScript/TypeScript CodeQL 通过。
4. 原先 Security 失败原因为 `Dependency review is not supported on this repository`，**不是实际发现高危依赖的结论**。不得通过删除 dependency-review job 或增加 `continue-on-error` 获取表面绿色。

## 2. 配置 preview Ruleset 或 Branch protection

在 [Settings → Rules → Rulesets](https://github.com/WatchTestCs/smart-elderly-watch-management/settings/rules) 选择新建分支规则，目标**仅** `preview`，规则启用前先检查所有需要的检查名称已在实际 PR 上出现。

必须启用：

- 合并只允许通过 Pull Request，阻止直接推送（视所选 ruleset 能力），禁止强制推送和删除分支。
- 要求状态检查成功后方能合入，检查在最新 HEAD 或其 GitHub 测试合并提交上通过；新提交会使旧检查结果失效。
- 推荐至少要求 `Preview CI / required`、`Preview checks / verify`、`UI Quality / verify`、`Desktop Contract`、`Windows unsigned desktop build` 及 `Security` 内 CodeQL/Dependency review 的实际稳定 job 名称。**创建规则时以 GitHub UI 的准确名称为准**，不要自定义不存在的 job 字符串；对不适用的文件/PR 可使用专门汇总检查避免永久 pending。
- 管理员和自动化账号的绕过权限尽量收紧；不应将「所有人都可绕过」作为默认策略。
- 若团队有独立代码审查者，要求至少 1 位有权限的审查者批准并在新增提交后撤销过期批准；如果项目只有一位贡献者，先确认审核工作流可执行，避免不可满足的永久锁定。

在启用前以一个草稿 PR 检查规则确实会阻挡合并；不能只看设置页面是否显示绿色。

## 3. 合并门槛与发布门槛分离

**Merge-ready 的自动化条件：**

- 所有适用的单元、集成、浏览器和构建检查通过；PR 无未解决的实质性问题。
- 涉及 Flyway 的 PR 在真实数据的**脱敏备份副本**上完成升级、失败恢复、重复运行和旧版拒绝测试，并归档哈希与关键证据。
- 涉及设备身份、SOS、告警的 PR 验证跨角色权限、真实设备协议兼容以及断网/重复/异常数据回归。
- CI 不可达或被 GitHub 配置阻塞时，标记 blocked，不手动勾选为 pass。

**Release-ready（高于 Merge-ready）：**

- 在真实 Windows 10/11 上测试未签名/签名安装、启动、升级、卸载与用户数据保留；
- 正式签名须有受信任证书及时间戳；未签名 `岁宁-Setup-2.0.0.exe` 的 CI 构建成功**不是受信任发布**；
- 真实手表、短信、外网地图和关键告警通路有实际运行证据，否则不得宣称完整发布验收。

## 4. 变更与验收记录

每次调整本规则：记录规则截图/规则 ID、影响分支、适用检查名称、当前授权绕过名单、PR 链接、验证时刻和验证结果。不能提交密钥、客户数据或私有凭据。至少每季度核对一次门禁是否因工作流改名、功能停用、取消运行而失效。

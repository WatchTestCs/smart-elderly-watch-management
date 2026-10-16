# Preview 发布流程

本文定义 `preview` 的 release candidate 和 GitHub prerelease 门槛。它描述“发布前必须证明什么”，而不是要求每个普通 PR 都执行完整 Windows 打包。

## 1. 发布输入

发布输入必须是：

- `preview` 上一个明确 commit SHA；
- 核心 CI 通过；
- UI Quality 通过；
- Desktop Contract 通过；
- Security 中 CodeQL 检查通过；
- 版本源 `desktop/version.json` 已确认。

禁止以工作区未提交文件、未知 commit 或本机临时补丁作为正式发布输入。

## 2. 阶段

```text
preview commit
  -> core CI
  -> UI quality
  -> security checks
  -> Windows desktop build
  -> unsigned application
  -> release approval
  -> signed application
  -> signed installer/uninstaller
  -> install / overwrite-upgrade / uninstall smoke
  -> checksums + SBOM + provenance
  -> GitHub prerelease
```

每一阶段失败都应停止后续阶段。

## 3. GitHub unsigned candidate

`.github/workflows/preview-candidate.yml` 是仅手动触发的 Windows 候选构建。它故意不接触签名私钥，负责：

- 运行真实 `desktop/build.ps1`；
- 生成未签名安装器与 Portable ZIP；
- 生成 SHA-256；
- 使用 Syft/Anchore 生成 SPDX JSON SBOM；
- 使用 GitHub `actions/attest` 为候选资产建立 SLSA build provenance；
- 为 Portable 资产建立 SBOM attestation；
- 上传短期 Actions artifact 供验收。

GitHub 的 artifact attestation 用于证明“哪个 workflow、仓库、commit 和事件生成了该资产”，不是对资产本身安全性的担保。

## 4. 本地 release candidate

```powershell
cd 智慧养老手表管理系统
./desktop/release.ps1
```

已有 build：

```powershell
./desktop/release.ps1 -SkipBuild
```

默认候选位于：

```text
outputs/v<version>-<channel>/
```

## 5. 必须存在的资产

至少包括：

- `installer/岁宁-Setup-<version>.exe`
- `installer/岁宁-Portable-<version>.zip`
- `evidence/SHA256SUMS.txt`
- `evidence/release-manifest.json`
- 许可/第三方 notices 随程序包存在。

使用本地自签名流程时，还会产生公开证书 `Suining-Local-Signing.cer`；它不含私钥。

## 6. Windows smoke

正式 preview 发布前必须在打包后的真实程序而不是 Vite dev server 上验证：

- 首次设置；
- 登录；
- 核心页面打开；
- 模拟设备上报；
- 退出时 Java 后端结束；
- 单实例；
- 重启后数据保留。

安装版额外验证：

- 静默/交互安装；
- 同版本或目标升级路径覆盖安装；
- 运行中程序阻止覆盖；
- 卸载保留 `%APPDATA%\岁宁`；
- 快捷方式与卸载注册信息正确。

## 7. 签名边界

普通 PR CI 不读任何代码签名私钥。

当前 `CN=Suining Local Build` 是本地自签名，仅证明产物签名在密码学上存在，并不能建立公众信任链。

若未来接入真实代码签名证书：

- 单独使用受保护 release environment；
- 最小化 secret 权限；
- 签名发生在 build 完成后；
- 签名前后的 hash 与 provenance 都要记录；
- 不允许 fork PR 调用签名步骤。

## 8. 发布证据

`release-manifest.json` 至少记录：

- product；
- version；
- channel；
- source commit；
- UTC 生成时间；
- 每个主要资产的 SHA-256 和 size。

GitHub prerelease 最终还应附加：

- SPDX SBOM；
- 可由 `gh attestation verify` 核验的 artifact attestation/provenance；
- CI run 链接；
- 已验证/未验证能力说明。

## 9. 已知不能省略的限制说明

只要这些事实仍成立，release notes 必须保留：

- Windows 10 未实机验证；
- 真实手表端到端联调未完成；
- 真实短信发送未完成；
- 没有正式性能基准；
- 没有完整第三方渗透测试；
- 自签名不是公众受信任代码签名。

## 10. 发布后

发布后：

1. 确认 tag 指向预期 preview commit；
2. 下载 GitHub Release 资产重新核对 SHA-256；
3. 确认 Release 页面没有数据库、私钥、日志、本机绝对路径；
4. 不为“修一个资产”移动既有 tag；
5. 若资产错误，发布新的明确版本/预发布序号并说明 supersede 关系。


## 11. 供应链自动化边界

- `.github/workflows/security.yml`：在 `preview` push/PR 上运行 CodeQL；PR 还运行 Dependency Review。
- `.github/dependabot.yml`：版本更新 PR 明确以 `preview` 为目标。
- GitHub 的 `target-branch` 只改变 Dependabot **version updates**；security update PR 属于独立机制，GitHub 会以默认分支为目标，因此当前不把它描述成 preview 专用能力。
- `.github/workflows/scorecards.yml`：OpenSSF 官方 Action 已预置，但官方对 push/schedule 要求默认分支。仓库默认分支仍为 `main`，所以 preview 不自动运行 Scorecard；当该 workflow 被有意提升到默认分支后再启用自动触发和公开结果。

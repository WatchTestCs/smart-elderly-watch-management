# Runbook: Preview Release

1. 确认 HEAD 位于预期 `preview` commit，工作区无未提交改动。
2. 确认 Preview CI、UI Quality、Desktop Contract 通过。
3. 核对 `desktop/version.json`。
4. 运行 `desktop/release.ps1`。
5. 核对 `evidence/release-manifest.json` 与 `SHA256SUMS.txt`。
6. 在隔离的 Windows 用户数据目录执行真实应用 smoke。
7. 安装、覆盖升级、卸载并确认用户数据保留。
8. 重新计算待上传资产 SHA-256。
9. 创建新的 prerelease tag，指向同一个 preview commit。
10. 上传 Setup、Portable、checksum、SBOM/provenance（当流水线提供）。
11. 从 GitHub Release 重新下载并再次核对 hash。
12. 在 release notes 明确列出未验证能力。

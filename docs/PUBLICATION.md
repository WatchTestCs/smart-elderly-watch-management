# preview 分支发布约定

本仓库当前工程化工作只操作并提交 `preview` 分支。保留 `main` 与其他分支，不强推，不改写既有历史，也不自动创建或合并 PR。

已有 `v2.0.0-preview.1` 属于既有预发布记录；后续发布不应通过移动该 tag 改写历史。新发布必须使用新的明确 tag，并精确指向对应的 preview commit。

## 发布资产

源码树不提交 EXE、完整运行包、用户数据库、私钥、构建缓存或验收数据库。

release candidate 由分层脚本产生：

- `desktop/build.ps1`：未签名应用；
- `desktop/package.ps1`：安装器打包；
- `desktop/release.ps1`：签名、Portable ZIP 与证据。

输出使用版本化目录 `outputs/v<version>-<channel>/`，不再使用日期硬编码目录。

## 可追溯性

GitHub Releases 中的主要资产必须有 SHA-256，并能从 `release-manifest.json` 追溯到唯一 commit。

后续自动发布还应加入 SBOM 与 artifact attestation/provenance。

## 签名

当前公开证书不含私钥。自签名不自动加入系统根信任，也不能描述为公众受信任发布者。

普通 PR/CI 不得接触签名私钥。

详细流程见 [RELEASE.md](RELEASE.md)；文件职责见 [FILES.md](FILES.md)。

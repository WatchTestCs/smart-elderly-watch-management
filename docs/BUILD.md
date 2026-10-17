# 开发与 Windows 打包

构建机器需 Node.js 22.12+、Java 17+（推荐 Java 21）、Maven 3.9、Python 3、PowerShell 7。最终用户不需要这些工具。

桌面构建已经拆成 **build → package → sign/release** 阶段。普通开发/CI 不再隐式创建签名证书，也不在一次脚本里混合所有职责。

## 版本源与输出目录

桌面版本元数据位于：

```text
智慧养老手表管理系统/desktop/version.json
```

当前：

```json
{
  "version": "2.0.0",
  "channel": "preview"
}
```

前端 `package.json` 与后端 `pom.xml` 的版本必须与该值一致；Windows Desktop Contract workflow 会自动检查。

默认输出：

```text
智慧养老手表管理系统/outputs/v2.0.0-preview/
├── tools/                 固定运行时/打包工具缓存
├── app-staging/           Electron ASAR staging
├── unsigned/
│   └── Suining/           未签名应用目录
├── signed/
│   └── Suining/           release 阶段复制并签名后的应用目录
├── installer/
│   ├── 岁宁-Setup-2.0.0.exe
│   └── 岁宁-Portable-2.0.0.zip
└── evidence/
    ├── uninstall-files.nsh
    ├── SHA256SUMS.txt
    ├── release-manifest.json
    └── Suining-Local-Signing.cer
```

不再使用包含具体日期的 `desktop-release-YYYYMMDD` 路径。

## 1. Build：生成未签名应用

从仓库根目录：

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
```

该阶段执行：

1. `npm ci`
2. Vite production build
3. Maven `clean verify`
4. packaging tools `npm ci`
5. 固定 Electron / JRE / NSIS 下载与摘要校验
6. Electron 封装与 ASAR/fuse 处理
7. 第三方许可清单整理

输出为 `unsigned/Suining`。

可以显式指定输出目录：

```powershell
./desktop/build.ps1 -OutputDir C:\temp\suining-build
```

## 2. Package：生成安装器

未签名内部包：

```powershell
./desktop/package.ps1
```

它使用 `unsigned/Suining` 构建安装器，但不会创建或使用签名证书。

也可以给定现有应用目录：

```powershell
./desktop/package.ps1 -OutputDir C:\temp\suining-build -AppDir C:\temp\suining-build\unsigned\Suining
```

## 3. Release：签名并生成可核验候选

本机 release candidate：

```powershell
./desktop/release.ps1
```

如果 build 已完成：

```powershell
./desktop/release.ps1 -SkipBuild
```

release 阶段：

1. 将 `unsigned/Suining` 复制为 `signed/Suining`；
2. 签名主程序；
3. 用已签名应用构建安装器；
4. 在 NSIS finalize 阶段签名卸载器；
5. 签名安装器；
6. 生成 Portable ZIP；
7. 生成 SHA-256；
8. 生成 `release-manifest.json`，记录版本、channel、commit、时间和资产摘要。

## 本地开发

前端：

```powershell
cd 智慧养老手表管理系统/oldIot
npm ci
npm run dev
```

开发代理连接 `127.0.0.1:8795`。

后端：

```powershell
mvn -f 智慧养老手表管理系统/careold/pom.xml spring-boot:run
```

首次管理员设置依赖桌面私有令牌。不要为了浏览器调试给设置接口添加公开默认账号或绕过桌面 token。

## 签名

`sign.ps1` 当前查找 `CN=Suining Local Build`；不存在时创建不可导出的本机代码签名私钥。

该证书：

- 是自签名；
- 不自动添加系统根信任；
- 不把私钥放入仓库或安装包；
- 不能被描述为公开 CA 信任的商业签名。

公开分发若需要受信任发布者，应替换签名实现，而不是把本地私钥上传到普通 CI secret 后继续沿用当前脚本。

## 自动检查

`.github/workflows/desktop-contract.yml` 在桌面构建相关文件变化时运行 Windows 检查：

- 前端/后端/桌面版本一致；
- 所有 PowerShell 文件可解析；
- Electron Node 脚本语法可解析；
- Python 构建脚本可编译；
- 活跃桌面脚本中不存在旧的 dated output path。

完整 build/package 仍属于 Windows release/smoke 阶段，不应在每个普通 PR 上重复下载大型运行时。

## 提交与分发

当前工程化范围仅限 `preview`：

- 不推送 `main`；
- 不自动合并；
- EXE/ZIP 放 Releases，不放源码树；
- 标签必须指向唯一 preview commit；
- 用户数据库、原始人员 SQL、私钥、构建缓存和验收数据库不得进入源码包。

发布流程见 [RELEASE.md](RELEASE.md)。

## 待合并修复的附加验证入口

### 版本迁移与数据恢复

后端新增 `careold/src/main/resources/db/migration/` 下的 Flyway V1/V2 及 `DatabaseMigrationTests`。升级既有本机 H2 文件数据库前必须验证离线备份、旧库迁移、重复日记录归档、重复启动和未来版本拒绝；迁移失败禁止清库和盲目修改历史脚本，参见 [问题清单核验](PREVIEW_ISSUE_REMEDIATION.md)。

### 浏览器与脚本测试

```powershell
cd 智慧养老手表管理系统/oldIot/qa
npm ci
npx playwright install chromium
npm test
cd ../..
node --test desktop/runtime-files.test.cjs watch-sim/watch-sim.test.cjs
```

QA 依赖只用于验收，不进入最终桌面运行包。

### 正式发布签名

本分支的 `desktop/sign.ps1` 支持 `SUINING_SIGNING_THUMBPRINT`、`SUINING_TIMESTAMP_SERVER` 与 `SUINING_REQUIRE_TRUSTED_SIGNATURE=1`。普通 `build.ps1` 仍只执行未签名构建；签名需进入受控 `release.ps1` 阶段。显式选择正式证书失败不可降级成自签后继续发布；正式证书仍需在可访问私钥与受信任环境中实测。

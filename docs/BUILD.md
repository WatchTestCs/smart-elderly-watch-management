# 开发与 Windows 打包

构建机器需 Node.js 22.12+、Java 17+（推荐 Java 21）、Maven 3.9、Python 3、PowerShell 7。最终用户不需要这些工具。

从仓库根目录执行：

```powershell
cd 智慧养老手表管理系统
./desktop/build.ps1
```

脚本顺序为 npm ci → Vite 生产构建 → Maven 测试与打包 → 固定工具/运行环境校验 → Electron 封装 → 许可整理 → 应用签名 → NSIS 安装器及卸载器签名。首次下载需要网络。当前输出在项目内 outputs/desktop-release-20261015/release。

## 本地开发

在 oldIot 运行 npm ci 与 npm run dev。开发代理连接 127.0.0.1:8795；在 careold 运行 mvn spring-boot:run。首次管理员设置仅接受桌面私有令牌，建议在打包窗口完成首次配置，而不是给网页增加公开默认账号。

## 签名

默认 sign.ps1 查找 CN=Suining Local Build；不存在时创建不可导出的本机代码签名私钥。该证书自签，不添加系统根信任，不把私钥打入包。正式签名使用证书提供方签发、已导入 Windows 证书存储且能访问私钥的代码签名证书：

```powershell
$env:SUINING_SIGNING_THUMBPRINT='你的代码签名证书指纹'
$env:SUINING_TIMESTAMP_SERVER='证书提供方给出的时间戳服务地址'
$env:SUINING_REQUIRE_TRUSTED_SIGNATURE='1'
./desktop/build.ps1
```

应用、安装器和卸载器共享上述配置。显式证书无效、缺少时间戳或验证失败会中止构建，不回退自签。公开 CER 从安装器的实际签名者导出，不包含私钥。用户目前没有正式证书，故这条正式签名路径尚未实签验收；签名有效不保证 SmartScreen 已有信誉。

## 数据库迁移与本地验收

Flyway 从 `careold/src/main/resources/db/migration` 管理版本；不要修改已经发布的迁移文件，也不要使用 clean 或手工改版本号处理生产数据库。升级前文件备份与未来版本保护见 [问题清单修复记录](PREVIEW_ISSUE_REMEDIATION.md)。

在 oldIot 运行 npm ci、npm run build；在 careold 运行 mvn clean verify。浏览器验收工具独立放在 oldIot/qa：运行 npm ci、npx playwright install chromium、npm test；Linux CI 安装浏览器时加 --with-deps。桌面/模拟器本地测试为项目根的 `node --test desktop/runtime-files.test.cjs watch-sim/watch-sim.test.cjs`。QA 依赖不进入产品包，结果写入忽略的 artifacts。

preview 专用 CI 文件已准备，但本轮没有推送或运行远程 Actions。本地测试结果与未执行项见 [本轮结果](preview-remediation-validation.json)。

## 提交与分发

该仓库仅提交 preview，不推送 main，不自动合并。EXE 和完整运行包放 Releases，不放 Git 源码树。预发布标签应精确指向 preview 提交。用户数据库、原始人员 SQL、私钥、构建缓存与验收数据库不得进入新源码包。

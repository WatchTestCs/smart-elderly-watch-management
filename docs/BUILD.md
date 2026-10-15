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

sign.ps1 查找 CN=Suining Local Build；不存在时创建不可导出的本机代码签名私钥。该证书自签，不添加系统根信任，不把私钥打入包。需要公开 CA 信任时替换签名流程使用真实证书。

## 提交与分发

该仓库仅提交 preview，不推送 main，不自动合并。EXE 和完整运行包放 Releases，不放 Git 源码树。预发布标签应精确指向 preview 提交。用户数据库、原始人员 SQL、私钥、构建缓存与验收数据库不得进入新源码包。

# 岁宁桌面源码工程

本目录包含当前 Windows 预览版的服务、界面、桌面封装与设备联调工具。

- [图文首页与软件下载](../README.md)
- [使用与数据备份](../docs/SETUP.md)
- [架构与优化说明](../docs/ARCHITECTURE.md)
- [完整构建指南](../docs/BUILD.md)
- [逐文件介绍](../docs/FILES.md)
- [已执行验收与边界](../docs/VALIDATION.md)

## 目录

careold 为 Spring Boot 3.5 / JDBC / H2 业务服务；oldIot 为 Vue 3 / Element Plus / Vite 8 界面；desktop 负责 Electron、随包 Java、完整性校验、安装和签名；watch-sim 提供模拟手表请求。

## 本目录构建

开发机器准备 Node.js 22.12+、Java 17+（推荐 Java 21）、Maven 3.9、Python 3 与 PowerShell 7，然后执行：

```powershell
./desktop/build.ps1
```

本地输出位于 outputs/desktop-release-20261015/release；GitHub 预览包通过仓库 Releases 分发，不把 EXE、构建缓存或用户数据库提交到源码树。

最终用户无需开发工具。首次打开软件创建自己的管理员账号，没有默认手机号和密码。数据默认位于 %APPDATA%\岁宁，升级和卸载保留；旧 MySQL 不自动迁入。

应用与安装/卸载程序为本机自签，不具备公开 CA 信任，不自动修改系统根信任。只提交 preview 分支。

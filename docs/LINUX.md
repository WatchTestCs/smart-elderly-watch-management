# Linux 软件与数据库管理工具

本次按用户提供的二进制附件补充到 2.0.2 Release，不将它们描述为从本仓库当前提交构建的 Linux 版本。原始附件保留不变。

## 岁宁软件包

`Suining-Linux-amd64.tar.gz` 与用户的“岁宁-linux-软件包.tar.gz”内容完全一致。内含 x86-64 ELF 服务程序、三个迁移 SQL、图标、启动/停止脚本及桌面入口。启动脚本运行本机 Go 服务并调用系统浏览器，是本地服务配合浏览器界面的交付形式，并非 Tauri 原生 Linux 桌面包。

解压后进入“岁宁”目录，按实际部署设置 `SUINING_DATABASE_URL`，运行 `启动岁宁.sh`；停止使用 `停止岁宁.sh`。启动脚本默认端口 8080，可通过 `SUINING_PORT` 调整。脚本需要 Bash，打开界面需要系统浏览器及 `xdg-open`。不要将真实数据库密码写入公共文档或源码。

该附件携带三个迁移文件，本仓库 Windows 当前后端有后续迁移；不能推定两者源码、数据库版本和安全修复完全一致。生产部署前应先备份并单独核对数据库兼容性。本轮未执行 Linux 程序或生产库迁移。

## 数据库管理工具

`suining-db-manager_1.0.0_amd64.deb` 为用户提供的数据库管理工具，组件版本仍为 1.0.0。它用于连接 PostgreSQL，不包含 PostgreSQL 服务或业务数据库，不会把数据库改成内嵌模式。

在支持包内依赖的 Debian/Ubuntu amd64 桌面系统，可使用：

```sh
sudo apt install ./suining-db-manager_1.0.0_amd64.deb
```

声明依赖为 `libwebkit2gtk-4.1-0` 和 `libgtk-3-0`；这些是 Linux 界面组件，与 Windows WebView2 不同。安装可能需联网取得发行版依赖。

原附件的 Debian `Package` 字段使用了中文，`dpkg-deb` 拒绝解析。发布副本仅将此字段修正为 `suining-db-manager` 并重打控制归档；程序所在 `data.tar` 成员逐字节保持不变。原始附件未修改。

## 本轮检查范围

仅检查压缩包目录、ELF 架构、DEB 元数据、保留程序负载与发布摘要；没有在 Linux 桌面启动、安装或连接实际数据库。Windows 沿用上一轮已通过的安装及功能验收，本轮不重复完整测试。

# 环境与配置（手动启动方式）

日常使用建议直接用 `智慧养老手表管理系统/智慧养老管理系统.exe`（菜单式一键启动）。本文件给出手工方式，以及本机已经验证过的环境参数。

## 一、已验证的环境

| 组件 | 本机验证过的版本 / 位置 |
| --- | --- |
| 数据库 | MySQL 8.0，服务名 `MySQL2077`，账号 `root` / `123456`，库名 `old`，端口 3306 |
| JDK | JDK 17（`C:\Program Files\Java\jdk-17`） |
| 后端 | Spring Boot 1.5.19，端口 8795，配置在 `careold/src/main/resources/applicationContext.xml` |
| 前端 | Vue 2 + Webpack 3，**Node 10**（项目内 `tools/node-v10.24.1-win-x64`），端口 8087 |
| Maven | 需自带 Maven 3.9.x（离线可用即可），`careold/mvnw` 缺少 `.mvn/wrapper`，不能直接用 |

## 二、数据库

1. 建库 `old`（字符集用默认即可）。
2. 导入 `old.sql`。脚本含 `DROP TABLE IF EXISTS`，会删除同名表，**不要导入已有业务数据的库**。
3. 老库升级（曾经导入过更早版本时）：依次执行 `invite_code_migration.sql`、`device_migration.sql`、`alert_handled_migration.sql`、`alert_handled_at_migration.sql`；这些脚本可重复执行，也可以直接用 EXE 的菜单 `[5]`。
4. 数据库地址、账号、密码在 `careold/src/main/resources/applicationContext.xml` 里改。

> MySQL 8 的 `ONLY_FULL_GROUP_BY` 默认开启会让老 SQL 报错，当前代码里的取数语句已经重写，不再依赖关闭该模式。

## 三、后端

在 `careold/` 下编译打包（示例为本机路径，可按实际调整）：

```powershell
$env:JAVA_HOME = "C:\Program Files\Java\jdk-17"
E:\DS\work\tools\apache-maven-3.9.9\bin\mvn.cmd -o -DskipTests package
```

启动（JDK 17 跑 Spring Boot 1.5 必须加 `--add-opens`，否则启动即失败）：

```powershell
java --add-opens java.base/java.lang=ALL-UNNAMED `
     --add-opens java.base/java.lang.reflect=ALL-UNNAMED `
     --add-opens java.base/java.util=ALL-UNNAMED `
     --add-opens java.base/java.io=ALL-UNNAMED `
     --add-opens java.base/java.net=ALL-UNNAMED `
     --add-opens java.base/java.nio=ALL-UNNAMED `
     --add-opens java.base/sun.nio.ch=ALL-UNNAMED `
     -jar careold\target\careold-0.0.1-SNAPSHOT.jar
```

注意事项：

- **打包含义**：正在运行的 jar 会被 JVM 锁住，重新打包会失败并把 jar 写成空壳。打包前先停掉后端进程。
- JDK 8 环境则无需 `--add-opens`。

## 四、前端

```sh
cd oldIot
npm install --ignore-scripts      # 老依赖的编译脚本在现代 Node 上会失败，跳过
node node_modules/webpack-dev-server/bin/webpack-dev-server.js --inline --progress --config build/webpack.dev.conf.js
```

- 必须用 Node 10；高版本 Node 跑 Webpack 3 会报 `ERR_OSSL_EVP_UNSUPPORTED` 之类的错误。
- 开发代理在 `oldIot/config/index.js`，接口目标 `http://localhost:8795/`。
- 生产构建入口是 `npm run build`，输出到 `oldIot/dist/`。
- `index.html` 中的地图脚本需要使用自己的地图 Key。

## 五、账号与网络

- 演示账号见 `系统使用手册.md` 第四节（管理员 `15179158704` / `123456`）。
- 后端默认监听全部网卡（`::8795`），手表 / 手机与电脑在同一 Wi-Fi 时可用电脑局域网 IP 访问；Windows 防火墙需放行 8795（TCP 入站）。

## 六、仍然属于演示范围的部分

- `applicationContext.xml` 中的数据库密码、短信 AppKey、地图 Key 都是演示值或占位符，公开部署前必须替换为自己的配置，不要提交新凭据。
- 短信 / 地图 / Solr / HBase 等外部服务只是依赖声明，未做过联调验证。
- 前端为 Webpack 3 老工具链，仅保证 `npm run dev` / `npm run build` 在本机 Node 10 下可用。

## 七、环境自检与「Git 不见了」的恢复（2026-10-13 实测）

体检脚本（复制到 PowerShell 里逐条跑即可）：

```powershell
git --version                                   # 期望 git version 2.53.0.windows.2
where.exe git                                   # 期望 D:\OPENCLAW\Git\cmd\git.exe
& 'C:\Program Files\Java\jdk-17\bin\java.exe' -version
Get-Service MySQL2077 | Select-Object Status    # 期望 Running
Test-Path '智慧养老手表管理系统\tools\node-v10.24.1-win-x64\node.exe'   # 期望 True
```

**症状**：`git` 报「无法将“git”项识别为 cmdlet…」，但 `where.exe git` 能打印出路径、或 PATH 里写着 `D:\OPENCLAW\Git\cmd` 而该目录不存在。原因是 Git 的安装目录被清理掉了（本机实测发生过一次；Git 装在自己的目录里，不在 Program Files）。

**恢复（一条命令，约 30 秒）**：

```powershell
# 安装包本机已有：E:\download\Git-2.53.0.2-64-bit (1).exe（另有一份在 C:\Users\Administrator\Downloads）
& 'E:\download\Git-2.53.0.2-64-bit (1).exe' /VERYSILENT /NORESTART /NOCANCEL /SP- /SUPPRESSMSGBOXES /DIR='D:\OPENCLAW\Git'
```

装到 `D:\OPENCLAW\Git` 是刻意选择：系统 PATH 里已经有这项，装回原位就不用改环境变量，也不需要重启电脑。

**恢复后验证**：

```powershell
git --version                                   # 2.53.0.windows.2
git -C 'C:\Users\Administrator\Desktop\智慧养老手表管理系统' fsck --no-progress   # 没有输出即仓库完好
git -C 'C:\Users\Administrator\Desktop\智慧养老手表管理系统' status --short       # 干净
& 'C:\Users\Administrator\Desktop\智慧养老手表管理系统\立即同步到GitHub.exe' --yes # 能提交并推送
```

凭据存在 Windows 凭据管理器里，重装 Git 后不用重新登录；本机 `git config --local credential.helper` 已是 `manager`。
另外，`立即同步到GitHub.exe` 会依次在 `D:\OPENCLAW\Git`、`C:\Program Files\Git`、`C:\Program Files (x86)\Git`、`%LOCALAPPDATA%\Programs\Git` 里找 `git.exe`，最后才交给 PATH —— 所以即使 PATH 被改坏，同步工具通常仍能用。

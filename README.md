# 智慧养老手表管理系统

基于 Spring Boot 和 Vue 2 的智慧养老手表管理项目，提供管理端和家属端页面，涉及老人档案、家属关系、手表设备、健康数据、位置、报警、公告与留言。

项目按源码整理；下述功能和步骤尚未经过完整部署及真实手表联调验证。

## 功能模块

- **管理端**：老人档案、家属和监护人、手表设备、报警、公告动态、留言，以及心率血压、睡眠步数和位置记录。
- **家属端**：查看关联老人的资料、健康记录、位置和公告动态。
- **后端**：Java Web 接口、Shiro 登录与权限配置、JDBC 数据访问、图片上传和下载。

上述内容根据现有页面、控制器和数据访问代码归纳；并非所有功能都经过运行验证。

## 项目结构

```text
.
├── docs/                              项目说明、启动指引和逐文件清单
├── 智慧养老手表管理系统/
│   ├── careold/                       Spring Boot 后端
│   ├── oldIot/                        Vue 前端
│   ├── old.sql                        MySQL 表结构和随包演示记录
│   └── 运行说明.txt                   原始运行说明（GB18030 编码）
└── LICENSE
```

- [配置与启动](docs/SETUP.md)：环境要求、后端与前端启动步骤。
- [文件用途清单](docs/FILES.md)：221 个原始文件的独立介绍及 SHA-256。
- [整理与验证记录](docs/PUBLICATION.md)：源码整理范围和当前验证边界。

## 技术组成

| 部分 | 技术 |
| --- | --- |
| 后端 | Java 8、Spring Boot 1.5.19、Spring JDBC、Apache Shiro |
| 前端 | Vue 2、Vue Router、Vuex、Element UI、Axios、ECharts、Webpack 3 |
| 数据与服务 | MySQL；源码另声明 Solr、HBase/Hadoop 和腾讯云短信依赖 |

前端依赖包含 `node-sass 4` 等较旧工具，在当前 Node.js 环境中的兼容性尚未验证。

## 快速启动

### 1. 准备数据库

准备 JDK 8、Maven、Node.js/npm 和 MySQL。`old.sql` 含有 `DROP TABLE IF EXISTS`，会删除数据库中同名表；**请先创建专用测试数据库，不要导入已有数据的数据库。** SQL 还保留了随包演示数据。

在 MySQL 中创建数据库并导入脚本：

```sql
CREATE DATABASE old CHARACTER SET utf8;
```

```sh
cd 智慧养老手表管理系统/careold
mysql -u root -p old < ../old.sql
```

### 2. 配置并启动后端

编辑 `智慧养老手表管理系统/careold/src/main/resources/applicationContext.xml`，设置本地 MySQL 地址、账号和密码。编辑 `application.properties`，把 `local-path` 改为存在且可写的本机图片目录。不要将自己的数据库密码或服务密钥提交到公开仓库。

```sh
cd 智慧养老手表管理系统/careold
mvn spring-boot:run
```

后端默认使用 `8795` 端口。

### 3. 配置并启动前端

先在 `智慧养老手表管理系统/oldIot/index.html` 配置自己的百度地图 Key，再启动前端：

```sh
cd 智慧养老手表管理系统/oldIot
npm install
npm run dev
```

默认访问 `http://localhost:8087/`。开发代理配置在 `oldIot/config/index.js`，默认将接口请求转发至 `http://localhost:8795/`。

## 部署前需要配置

- **MySQL**：项目数据源在 `applicationContext.xml` 中配置。
- **图片存储**：`application.properties` 中的 `local-path` 必须指向可写目录。
- **百度地图**：前端地图 Key 需要自行申请和配置。
- **Solr、HBase、短信和消息中间件**：源码包含相关依赖或地址配置，按实际使用的功能配置对应服务；仅有依赖不代表服务已包含或已经验证。
- **凭据**：仓库中的短信 Key 和地图 Key 是占位符；数据库连接等遗留设置也应在本机检查并更换。

更多说明见[配置与启动指南](docs/SETUP.md)。目前没有声明构建通过、生产部署就绪或真实手表连接成功。

# 智慧养老手表管理系统

基于 Spring Boot 和 Vue 2 的养老手表管理项目，包含 Java 后端、管理及家属端页面、MySQL 脚本与静态图片。

保留原压缩包内全部 221 个文件及目录名，按文件提供者要求清理作者注释、包作者邮箱、旧服务器地址和指定手机号。未删除文件、重构业务或升级依赖。新增文档放在外层，与工程分开。所有功能说明依据源码和随包资料整理，尚未完成运行或设备联调验证。

## 目录与介绍

| 路径 | 内容 |
| --- | --- |
| `智慧养老手表管理系统/careold/` | Java 后端，含控制器、JdbcTemplate 数据访问与领域对象 |
| `智慧养老手表管理系统/oldIot/` | Vue 前端，含管理端、家属端页面及静态资源 |
| `智慧养老手表管理系统/old.sql` | MySQL 转储，13 张表与 137 条 INSERT 语句；指定账号已同步替换 |
| `智慧养老手表管理系统/运行说明.txt` | 运行说明，保持原始 GB18030 编码；指定账号已替换 |
| [docs/FILES.md](docs/FILES.md) | 每个原文件的名称、用途、大小和 SHA-256 |
| [docs/SETUP.md](docs/SETUP.md) | 根据源码整理的环境、配置与启动说明 |
| [docs/PUBLICATION.md](docs/PUBLICATION.md) | 保留范围、敏感内容与验证记录 |
| [docs/REPOSITORY.md](docs/REPOSITORY.md) | 建议仓库名称、简介和主题 |

## 功能范围

源码包含老人档案、家属与监护人、手表设备、心率血压、睡眠步数、位置、报警处理、公告动态、留言评论等相关模块。管理端与家属端使用不同页面及路由；代码存在不等于这些功能均已运行验证。

## 技术组成

- 后端：Java 8、Spring Boot 1.5.19.RELEASE、Spring JDBC、Shiro、MySQL Connector/J。pom 还声明了 Solr、HBase/Hadoop 和腾讯云短信 SDK。
- 前端：Vue 2、Vue Router、Vuex、Element UI、Axios、ECharts、Webpack 3、node-sass 4。
- 默认开发端口：前端 `8087`，后端 `8795`。

## 使用前

参阅 [环境与配置](docs/SETUP.md)。短信密钥和地图 Key 已替换为占位符；数据库密码与其他原始数据按全量保留要求保留，使用前请检查配置和数据。

本次完成归档整理、字节一致性和静态格式核对，未安装依赖、运行应用、导入数据库或连接真实手表。源码保留历史文件及旧依赖，不代表可直接用于生产部署。

原始压缩包没有 LICENSE；GitHub 仓库创建时已有 LICENSE，本次保留该文件。清理作者注释不代表重新确认了代码与图片的授权链。

短信 AppKey 和地图 Key 已替换为 `YOUR_TENCENT_SMS_APP_KEY` / `YOUR_BAIDU_MAP_AK` 占位符；这两项功能需自行配置后验证。

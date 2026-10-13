# 环境与配置

工作目录为 `智慧养老手表管理系统/`。以下来自源码配置和脚本，尚未运行通过。

## 后端

1. 准备 JDK 8 和独立 Maven。原包的 `mvnw` / `mvnw.cmd` 缺少 `.mvn/wrapper` 配套文件，不能认为 Wrapper 完整可用。
2. 在专用本地 MySQL 中创建 `old` 数据库，再导入 `old.sql`。脚本含 `DROP TABLE IF EXISTS`，会删除同名表，且包含原始人员与账号数据；不要导入已有业务数据的数据库。原转储标注 MySQL 5.0.22，pom 使用 Connector/J 8.0.20，实际兼容性待验证。
3. 检查 `careold/src/main/resources/applicationContext.xml` 的数据库地址、用户名和密码，替换成自己的本地配置，不要向版本库提交新凭据。
4. 检查 `application.properties` 的图片目录 `local-path`，创建目录并赋予应用写入权限。默认后端端口为 `8795`。
5. 按功能需要配置 Solr（默认 `http://localhost:8080/solr/test`）、HBase ZooKeeper（默认 `127.0.0.1:2182`）及短信 SDK；依赖声明不表示服务已验证可用。不要使用原包中他人的短信密钥。
6. 在 `careold/` 执行：

```sh
mvn spring-boot:run
```

## 前端

在 `oldIot/` 执行：

```sh
npm install
npm run dev
```

默认访问 `http://localhost:8087/`。开发代理位于 `oldIot/config/index.js`，接口目标为 `http://localhost:8795/`；地图脚本在 `oldIot/index.html`，需使用自己的地图 Key 与域名配置。

构建入口为 `npm run build`，输出目录 `oldIot/dist/`。这些步骤未运行验证；node-sass 4 / Webpack 3 等旧工具与现代 Node.js 的兼容性需另行确认，不将 package.json 的最低版本声明当作已验证环境。

## 账号、编码与完整性

原包账号说明保留在 `运行说明.txt`，本文不再重复账号与密码；账号能否使用、登录及角色行为均未验证。该 TXT 和 application.properties 含 GB18030 文本，查看时选择相应编码，原文件未转码。

原说明提到 MyBatis 和 Tomcat，但当前 pom 主要声明 Spring Boot / JDBC 依赖；以源码为准，不据此声称存在完整 MyBatis 实现或已验证外置 Tomcat 部署。

部署前还需验证数据库与查询是否匹配、外部服务是否完整，并检查权限控制、SQL 参数化、密码存储与文件上传。此归档不是生产验收结果。

## 本次清理后的地址与账号

旧图片服务器前缀已统一改为 `/restful/image/download?name=`，使用后端已有图片下载接口；开发时经前端代理访问。需在 `local-path` 下准备相应图片，图片实际存在与响应显示效果未联调验证。

消息中间件地址已改成本地 `http://localhost:8080/mqtt/request`，项目没有提供该中间件，使用时需按实际部署配置。原管理员电话形式账号统一改为演示编号 `10000000001`，数据库中关联引用同步替换，其他账号及原数据保留。

短信 AppKey 和地图 Key 已替换为 `YOUR_TENCENT_SMS_APP_KEY` / `YOUR_BAIDU_MAP_AK` 占位符；这两项功能需自行配置后验证。

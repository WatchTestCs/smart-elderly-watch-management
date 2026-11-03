# Go OpenAPI 与 TypeScript

`backend/cmd/openapi` 在内存中调用实际应用的 OpenAPI handler，输出与 HTTP 端点相同的
文档，不启动监听端口或连接数据库。前端使用固定的 `openapi-typescript@7.13.0` 生成
`src/api/generated-api.d.ts`，目前包含 109 条路径。该文件不得手工修改。

在 Go 位于 PATH 的开发环境中，从 `智慧养老手表管理系统/frontend` 执行：

```sh
npm ci
npm run api:generate
npm run api:check
npm run test:api-drift
npm run typecheck
```

`api:check` 重新导出 Go 文档并生成类型，以字节比较阻断陈旧文件；生成文件与状态定义
固定使用 LF，避免 Windows 检出时出现仅由换行导致的错误。Frontend CI 安装固定 Go
版本，并执行生成检查、反向编译检查、现有测试与类型检查；证据附源码 SHA。

依赖准备单独执行 `go mod download`、完整模块清单解析和 `go mod verify`。仅明确的
网络失败可在原阶段最多尝试三次，逐次保存日志；checksum、权限、版本或锁文件变更
直接失败，测试阶段离线运行且不自动重试。该处理针对 [23e99e8 的 push CI 失败](https://github.com/WatchTestCs/smart-elderly-watch-management/actions/runs/41631828650)：
模块代理在下载 x/text 时返回 HTTP/2 INTERNAL_ERROR，相关包在 setup 阶段即失败。
失败归档保留，同 SHA 的 PR 成功不会覆盖这次失败；后续候选仍需重新完整验证。

## 当前已接入的业务

最近健康采样使用 `postContract`，请求及响应均从生成类型获取，不再维护独立的
`LatestHealthRecord` 接口。传输仍使用原 Axios 实例，保持相对 API 地址、cookie、CSRF
拦截器和 AbortSignal；没有另起一条 fetch 认证路径。

该响应直接序列化 Go `health.Record` / `health.Sample`。OpenAPI 从相同 Go 类型导出
字段、nullable 和 omitempty，补齐原定义缺失的元数据、版本、分组等字段。未知 JSON
表示、未支持 tag、自定义序列化和递归 DTO 明确报错，需要审查专用 schema；不猜类型。
测试独立核对 encoding/json 的实际字段、必需值、未绑定 deviceId 和字符串元数据。

反向测试在内存中改变导出的文档，再生成类型并编译同一个真实客户端消费者，验证：

- 删除路由后，旧调用无法编译。
- oldId 从数值变为字符串后，原请求无法编译。
- 重命名采样 metric 后，原消费者无法编译。
- 元数据值由字符串变为数值后，原显示逻辑无法编译。

这些测试不会修改源码、更新基线或用注释忽略类型错误。客户端测试另核对原传输与取消
信号的委托行为。生成类型用于静态约束，不等同于运行时 JSON 校验或数据库集成测试。

## 尚未完成的范围

部分旧接口的 OpenAPI 仍仅有字段清单或开放对象，生成类型保留 `unknown`。109 条路径
进入生成链路不等于 109 条路径已完成详细 DTO、权限及异常合同验收。其余业务客户端
仍须逐步接入；P1-02、P1-07 的完整覆盖需要继续核查实现和测试，不能以生成文件大小
计算完成度。图片与 multipart 等非 JSON 传输也须按各自协议核对。

新增生成器为 MIT 许可开发依赖，不进入产品运行时。2026-11-03 的 npm audit 未报告
High/Critical；存在开发工具的 Moderate 间接依赖告警（含 Redocly/minimatch 和已有
测试工具链），纳入 P2-04 跟踪，不通过强行覆盖不兼容主版本掩盖告警。

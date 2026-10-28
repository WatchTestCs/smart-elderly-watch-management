# 前端覆盖与实施边界盘点

基线：`3b79b1f90e6cde6656fc40b9cee2e6af79ff0fe9`。本轮只新增基线工具、证据和交接；未修改应用依赖、业务实现、工作流或分支保护。任务负责人为当前执行人，无独立复核人。

## T03：流程 → 脚本 → 触发与阻断

| 流程 | 实际入口 | 触发 | 代码中的交付阻断关系 | 本轮证据 |
| --- | --- | --- | --- | --- |
| 前端21项单测、类型、生产构建、依赖审计 | `reusable-frontend.yml`，由 `ci.yml` 调用 | preview PR；preview/codex/** push；手动 | CI required 的 needs 包含 frontend；失败使聚合失败。仓库管理员实际保护规则本轮未读取，不能由 workflow 推定 | `commands.json`、`unit.txt`、`typecheck.txt`、`audit.txt` |
| 登录可访问性、1440×1000 / 390×844 / 手机大字截图及粗粒度图像差异 | `ui-quality.yml` → `tests/ui/ui-smoke.mjs` | preview PR/push 且命中 paths；手动/被调用 | 独立 workflow；主 CI needs 与 release-gate.py 未包含它。不能称为现有 required 的完整业务覆盖 | 基线 UI Quality run 40511021534 success；本机 qa 并非此远端脚本的替代结果 |
| 浏览器多路由、登录登出、伪造存储/家属权限、表格筛选/分页、表单失败保留、地图授权/失败、生命周期 | `oldIot/qa/run.cjs`，顺序11组脚本 | `preview-checks.yml` 仅 workflow_dispatch，且只在 preview 执行 | 手动流程内非零即失败；没有被主 CI/发布聚合依赖。codex 分支 push 不执行它 | 本轮实际执行全部11组，1386个断言通过；所有 API mocked |
| 会话过期、主库不可用、恢复保留输入、不重放写入 | `qa/database-availability-acceptance.cjs`，包含于上行 | 同上 | 同上，当前不是主 CI 强制门禁 | 13个断言通过；以 HTTP mock 切换状态，不能称为真实数据库断开/恢复 |
| 管理职责、禁止自我复核、事件分页/历史/失败状态 | `qa/architecture-acceptance.cjs`，包含于 qa/run | 同上 | 同上 | 23个断言通过，mock |
| 实际登录→监护人/老人建档→设备绑定→阈值更新→心率缺测→SOS处置→第二人独立复核/历史 | `qa/real-backend-acceptance.cjs` | 需自行启动隔离 Go/MySQL 并提供 CARE_E2E_*；qa/run 不包含它；相关 workflow 未发现调用 | 当前无自动必需门禁 | 本轮15项通过，Go内嵌当前生产前端，真实新建 MySQL schema、无 API mock；未做完整删除 CRUD，也未在该浏览器流程中停库恢复 |
| Go/手表协议/Windows sidecar/部署/安全 | `ci.yml` 多个 reusable workflows | preview PR；preview/codex/** push；手动 | 聚合 required 显式需要六组；release-gate.py 核对 CI、Security、聚合成功及六类 SHA artifact | 基线各 run success；不等于原生 WebView2 窗口或真机已运行 |
| 正式候选/准备发布 | `preview-candidate.yml`、`preview-release.yml` → exact-source/release-gate.py | preview 上手动、完整 source_sha | 必须当前 preview SHA，CI/Security 及六类 artifact 一致；安装器前后再核对 | 本轮未触发，不创建发布或更改管理员规则 |

1386为11组 JSON 内 `checks` 数量之和，包含表格/视口重复断言和 CSRF 断言，不代表1386条独立业务流程。真实后端15项另算。基线远端状态原文索引见 `evidence/frontend-20261028/baseline-actions.json`。

待接入的 T03 门禁：真实浏览器会话失效/权限/停库恢复、完整一次 CRUD 和故障可检出证明。接入前预登记注入点，证明失败会被需要它的 CI/job 聚合阻断；不能通过远端故意制造一次失败来代替本地故障注入。此次只有盘点完成，T03 整体未完成。

## T01/T05：模块清单

实际 `vue-tsc --listFilesOnly` 项目文件为9个：client.ts、latestRead.ts及其测试、useCustomers.ts及其测试、SecurityPanel.vue、RecordCards.vue、customer.vue、env.d.ts。RecordCards 是传递进入，不能只按 include 的字面数量统计。`checkJs=false`，不提供全仓 JS 语义保护。

| 模块 | 当前边界 | 下一批实施范围（G1-V放行后） | 剩余/验收 |
| --- | --- | --- | --- |
| 会话/权限 | `auth.js`、`router.js` 为JS；有会话/CSRF、401/403处理及不重放约束，但不在当前类型入口 | 先定义会话响应、角色/职责及 Axios 自定义 skipSession 类型，再增量检查相关文件 | 注入错误角色/字段与参数，证明旧漏过新阻断；401/403/CSRF和失效回归 |
| 设备 | `views/device.vue`、watch.vue 等仍直接使用 Axios；不在本轮TS入口 | 设备列表/查询 DTO，明确只读请求边界，先 device 的读取分支 | 不把轮询终止已有处理当作新增成果；绑定、创建等写入继续保留，不能套查询重试 |
| 健康 | ratepre/rateprefamily/step/stepfamily/position 页面多为JS直接Axios；ECharts core、ResizeObserver、dispose已存在 | 心率/血压列表与详情为首批；逐页验证空、缺失、null、非法类型 | 不能把缺测改为0或正常；其余睡眠/步数/位置未覆盖，不能用一个页面关闭整个T05 |
| 监护人 | customer.vue + useCustomers.ts 已有静态DTO与列表 envelope 部分运行时验证 | 复用现有 API/client；逐字段收紧列表值和变更响应合同；修复查询挂起 | 不另造 customers API 层；保留失败表单/删除记录、分页纠正、不得写入重放 |

`api-contract-inventory.json`：OpenAPI 有路由/鉴权/兼容别名；上述模块分别3/24/17/4个 legacy 操作的200响应均为开放 object，未给出精确响应字段与可空合同。现有生成器诚实标记未知 DTO，不能直接生成精确业务类型来冒充运行时校验。本轮不修改后端合同；后续逐项比对 handler 与真实返回，再决定补精确 schema。

## T02：已复现的机制风险

全局没有 axios.defaults.timeout；auth.js 的会话与数据库探针分别有5000ms，不能外推到所有业务查询。client.ts 只有 withCredentials，无 signal/timeout；latestRead 供 useCustomers 和 CareInbox 使用。

`query-hang-probe.mjs` 对旧源码三次注入永不完成的活动读取：新查询未被调用、busy保留true，dispose之后返回 Promise 仍不完成。释放旧 Promise 后无过期回写，正常邻接读取成功；既有21个单测也通过。这证明当前机制边界，未证明真实线上事故，更未修复它。

恢复实施时：仅为查询定义超时、取消和卸载；检查 signal 是否实际传至 Axios；测试不理会 abort 的旧请求也不回写、快速查询/卸载/新查询恢复、当前错误与取消区分。写入响应丢失要保留输入并核实服务器结果，不能自动重发。

## T04/T06/T07：资源、样式和静态规则

- element.js 已选择组件全局注册、中文 ConfigProvider、loading/Message/MessageBox；main.js 仍加载完整 Element Plus CSS。T04 尚未实验，不能凭 CSS 存在就说按需一定有效。将来先登记字节与比例门槛，再比较两轮构建和路由累计资源，不只看一个 chunk。
- `css-inventory.json`：element-theme 29条规则/7个!important，commercial 155条规则/1个!important。同 selector/同 property 跨文件直接重叠2处：dialog body padding（24px 28px → 小于等于600px为20px）及 footer padding（18px 28px → 16px 20px）。这是有媒体条件的响应式规则，当前 QA 通过，不能据此认定错误或直接删掉。
- 样式当前加载顺序：Element Plus → icons/fonts/index/tokens → element-theme → accessibility → commercial；具体输入边框、焦点、错误、禁用态已有 controls 656条断言。尚未完成全部选择器交集、跨具体性冲突与所有 teleport computed-style 记录，T06不关闭。
- T07：没有 lint script/ESLint配置/必需lint job。后续只给本批真实错误规则和文件范围，先以代表性 undefined variable 证明旧检查漏过、新检查阻断，再接必需入口，不引入全仓格式化。此次未新增lint实现。

## 原生与环境边界

机器：Windows 11 build22631.4169，i5-14600KF，14核20线程，约32GiB。Node24.22.1/npm11.6.2；Edge156.0.4314.42。只检测到 WebView2 155.0.4283.62 已安装，本轮没有运行原生窗口/IPC/桌面键盘验收。

真实后端使用独立工作树 `frontend-real-build-20261028` 固定基线，Go1.26.10构建，MySQL8.0.43端口33371，新建 `qa_arch_dd6e14ed187e`；运行后仅停止本轮拥有的 app 与 MySQL。原 MySQL2077 服务本轮末观测为 Stopped，本轮没有对该服务执行启停。原私有实验密钥/数据库配置均不提交。

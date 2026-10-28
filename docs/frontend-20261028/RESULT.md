# 2026-10-28 前端接续结果：基线与预检已完成，迁移受阻

交付分支 `codex/frontend-validation-20261028`；应用基线 `3b79b1f90e6cde6656fc40b9cee2e6af79ff0fe9`。已先保存 [PLAN.md](PLAN.md)，再执行、修正测量工具并验收。本轮交付测量工具、原始证据、任务盘点及交接；应用的 Vue/锁文件、网页、后端、桌面、手表业务代码和 workflow 均保持基线。没有将整个清单标记为完成，没有合并 preview/main 或创建发布。

## 已验证结果

| 检查 | 实际结果/证据 |
| --- | --- |
| 干净安装、完整依赖树 | npm ci/npm ls --all退出0；锁文件 SHA-256 `1a504ef004f882ae4182b4ace4e0faff64eb4f8c27294a2722127139f6b4a683`，前后相同 |
| 基础检查 | 6文件/21项单测、typecheck、两轮生产构建、npm audit全部退出0；audit为0漏洞。构建仍提示两个chunk超过500kB，此警告未隐藏 |
| 现有浏览器QA | 11组、1386条断言，全部通过；API均mock；各报告在 evidence 的 qa/ |
| 真实后端浏览器 | Go/MySQL合成测试库15项通过：建档/绑定/阈值/缺测/SOS独立复核/历史；不等于完整删除CRUD或停库恢复全部验收 |
| T00 | 登录、健康列表/图表、位置地图冷暖各5次；20轮图表开关固定点；资源raw/gzip/Brotli及路由累计清单；可复跑入口在 scripts/frontend |
| T02预复现 | 三次活动read悬挂均阻塞最新查询，dispose不结束活动Promise；现有旧结果屏蔽和正常邻接场景保留。未做修复 |
| T08依赖预检 | RC精确版本存在；3个直接依赖不接受其peer；默认lock-only生成退出0带覆盖警告，干净npm ci退出1 ERESOLVE；严格解析三次退出1。主项目未升级 |
| 包级回退 | 隔离目录用原package/lock做干净ci和ls均退出0，锁哈希相同；只是包级回退，未冒充Vue3.6应用完整回退 |

T00 的生产构建指 production bundle；测量服务采用固定测试缓存策略（静态资源max-age3600、HTML/API no-store、文本gzip），不是对线上服务器配置或性能的承诺。地图耗时到本地 Leaflet 标记，不包含真实供应商瓦片时延。无性能优化/迁移收益声明。

| 浏览器探索基线（ms，中位数 [min,max]） | 冷缓存页面就绪 | 暖缓存页面就绪 |
| --- | --- | --- |
| 登录页 | 98.31 [93.49,166.73] | 49.78 [44.24,55.39] |
| 家属健康列表 | 170.29 [164.94,209.50] | 88.44 [83.06,104.15] |
| 家属位置地图页 | 204.88 [163.69,236.16] | 122.38 [110.83,170.81] |

三场景每种缓存状态各5个样本。就绪终点为对应页面控件可见、字体加载完成和两帧；动作起止定义与原始timing/resource条目见 browser-baseline.json。20轮图表开关耗时中位59.66ms [48.55,247.87]；关闭后DOM实例标记和canvas均0；CDP监听器127–131，GC后堆约7.89–9.42MiB。这些是固定点观察，不证明没有所有内部泄漏，也不用堆增长断言泄漏。

| 页面就绪前累计JS/CSS | raw bytes | gzip9 bytes | Brotli11 bytes |
| --- | ---: | ---: | ---: |
| 登录 | 1160142 | 304057 | 250078 |
| 家属健康 | 1691363 | 482670 | 401330 |
| 家属位置 | 1333120 | 357432 | 296529 |

不包含字体/图片二进制、在线瓦片及详情动作之后新增请求；HTML另在完整资源清单。所有输出JS/CSS/HTML合计raw2145848 / gzip636199 / Brotli530332字节；gzip/Brotli是固定参数离线计算，实际HTTP传输另有原始条目，不混算。

## Vue迁移：明确阻塞与恢复条件

2026-10-28重查 [Vue官方RC发布](https://github.com/vuejs/core/releases/tag/v3.6.0-rc.13)、[minor更新日志](https://raw.githubusercontent.com/vuejs/core/minor/CHANGELOG.md)、[npm元数据](https://registry.npmjs.org/vue)：latest为3.5.44，rc为3.6.0-rc.13（2026-10-23发布、预发布）。日志中RC.13的vdom修正涉及异步加载延迟和离场中卸载；多数条目为Vapor。本项目不启用Vapor；尚未发现并复现能归因迁移修复的项目缺陷，不能把发布说明当作项目收益。

| 依赖 | 当前保留版本 | 官方Vue peer/Node engine | RC.13预检 |
| --- | --- | --- | --- |
| Vue Router | 4.6.4 | vue ^3.5.0 | 不接受RC |
| Element Plus | 2.14.7 | vue ^3.3.7 | 不接受RC |
| plugin-vue | 6.0.10 | vue ^3.2.25或^3.6.0-rc.10；Node ^20.19或>=22.12 | peer/Node范围接受；未等同于全应用验收 |
| test-utils | 2.5.1 | vue、compiler-dom、server-renderer均3.x | 稳定范围不接受该预发布版本 |
| vue-tsc | 3.3.13 | TypeScript>=5，无Vue peer | TS范围接受；迁移实测未执行 |
| TypeScript | 5.9.3 | Node>=14.17 | Node24满足 |
| Vite | 8.3.4 | Node ^20.19或>=22.12 | Node24满足，仍保留 |
| Vitest | 5.0.5 | Vite ^6.4/^7/^8；Node ^22.12/^24/>=26 | 当前Node24满足；项目engine写的Node20不满足此开发测试工具声明，未宣称Node20全链路支持 |

预发布不自动满足稳定范围，依据 [npm semver规则](https://github.com/npm/node-semver#prerelease-tags) 及本机实际包管理器结果。plugin-vue显式列出RC，因此不同。metadata/semver计算、默认解析、干净安装和严格解析原文均保留。原Vue九个核心包实际均3.5.44；未生成虚假的一致3.6安装树。

同时检查发行替代项：Element Plus和test-utils最新仍是现有版本；Router最新5.4.0的Vue peer仍未接受该RC，且引入额外工具约束，不能靠无关主版本升级解决。没有使用force、legacy-peer-deps、全局overrides、手改lock或删除测试包。

恢复：出现经过官方发布、peer接受的精确3.6及工具组合（或正式3.6）后重新核对当前preview、依赖与锁；再干净安装、全量适用回归和真实原生WebView2。T08仍须独立项目收益：性能按预登记10ms且5%的门槛、A/A及双批每版≥20独立样本、A/B/B/A和配对95%CI；或确定性正确性/实际必需能力的旧失败新通过证据。依赖可安装不自动放行。

## 复跑与证据

先安装项目锁定的Node兼容版本和npm；Windows可设置已有Edge路径，无需下载另一浏览器。示例（参数请替换为本机路径，证据目录必须尚不存在）：

```powershell
$env:CARE_CHROME_PATH='C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
node scripts/frontend/run-baseline.cjs <npm-cli.js绝对路径> <新证据目录绝对路径>
node scripts/frontend/migration-preflight.cjs <npm-cli.js绝对路径> <全新隔离实验目录> <含registry-metadata.json的证据目录>
node scripts/frontend/query-hang-probe.mjs <输出JSON绝对路径>
```

预检脚本退出0只表示收集完实验，必须读取 dependencyResolutionAccepted；此次是false。真实后端脚本另需新建UUID测试库及本地启动过程，当前私有运行器位于 `.reliability-lab/frontend-real-backend-20261028.py`，不提交其凭据来源。仓库内原qa脚本及真实后端环境/产物哈希可供复核，复跑时不得指向业务库。

`evidence/frontend-20261028/commands.json`、candidate-tree-commands.json记录具体命令和退出码；replay/是新增统一入口的再次实跑。早期同地址导航误采、about:blank引发localStorage错误、QA routing禁缓存的pilot和inventory工具解析错误均保留为无效/试采记录，不计合格样本。最终只使用 browser-baseline.json 与 replay/同名结果。完整哈希见 SHA256SUMS.json；检查代码/文件范围见 source-scope.json。

WebView2只检测到安装版本155.0.4283.62，未做原生UI；Edge156浏览器通过不替代原生。没有独立复核记录，不使用旧CI替代新候选CI。各T任务状态与M01–M08见 [TASKS.md](TASKS.md)，实现边界见 [INVENTORY.md](INVENTORY.md)。

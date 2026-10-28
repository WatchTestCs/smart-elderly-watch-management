# 2026-10-28前端原始证据

应用基线 `3b79b1f90e6cde6656fc40b9cee2e6af79ff0fe9`，任务计划/结论/边界在 `docs/frontend-20261028/`。原始文件按字节保存；所有文件（不含清单自身）的SHA-256见SHA256SUMS.json。数值只是基线，没有迁移收益或生产性能声明。

- environment.json、hardware.json、commands.json：环境、锁文件哈希、初次命令退出码；baseline-actions.json：当前应用基线的远端检查。
- browser-baseline.json：合格的HTTP fixture冷暖资源/页面/动作数据及20轮图表固定点；resource-sizes.json、route-resources.json：离线压缩和路由累计资源。
- replay/：新统一入口 run-baseline.cjs 的实际完整复跑，含两轮相同产物证明；独立合格样本，不把5次探索样本当迁移性能收益。
- qa/：既有11组mock及15项真实后端报告；qa-summary.json中分开；真实后端环境、构建、脚本输出另列；screenshots/仅合成数据。
- vue36-resolution-1/2/3.txt：默认lock-only生成虽退出0，但存在peer覆盖警告；不是安装通过。vue36-ci.txt：生成后干净ci失败。vue36-strict-1/2/3.txt：严格解析失败。candidate-tree-commands.json保留这些实际退出码。
- registry-metadata.json、dependency-alternatives.json、vue36-preflight.json：官方来源、版本约束和semver预检。dependencyResolutionAccepted=false。默认预检脚本退出0表示收集完成，不表示迁移成功。
- rollback-vue35*：只对原package/lock做隔离包级回退，ci/ls通过、锁不变；不冒充完整3.6业务回退。
- query-hang*：三次旧源码确定性机制复现与正常邻接，不是修复结果。
- typecheck-file-list.txt、api-contract-inventory.json、css-inventory.json：覆盖与合同/样式盘点，不是类型或样式实现完成。
- browser-baseline-initial-invalid*：同地址goto可能未触发文档导航，无效。
- browser-baseline-aboutblank-failed*：about:blank触发原harness的localStorage错误，失败，不能计入通过样本。
- browser-baseline-routing-cache-disabled-pilot*：文档导航有效，但Playwright routing禁止HTTP cache，不能作真正冷/暖缓存证据。
- inventory-initial-tool-failed.txt：首次假设postcss位于顶层导致工具失败，已改为从实际compiler-sfc依赖解析；保留失败原文。

测试缓存口径与线上服务器不同；真实后端与原生WebView2不同。当前报告没有真实供应商、人工接手、原生窗口或真机验收。应用/工作流未改，新增工具未接入必需CI，T03仍保留。

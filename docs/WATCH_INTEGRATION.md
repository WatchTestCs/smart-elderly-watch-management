# 手表端同分支整合计划与验收

本次用户要求将 `E:\DevEco Test\SuiningWatch` 的手表源码加入此前网页/后端所在的新分支，
并同步更新交接文档。起点为 `30464e55008149c8d3712d8f4d129eadaf7e80c3`，
目标仍为 `codex/reliability-validation-20261026`，不修改 preview/main。

## 执行计划（复制和修改前制定）

1. 读取原工程与忽略规则，确认配置、签名及生成目录；只复制源码、资源、构建配置、脚本与测试。
2. 在仓库根添加 `SuiningWatch/`，保留原独立目录；使用锁定 npm 测试依赖消除本机 IDE 路径依赖。
3. 运行协议测试、SDK 原生构建和独立测试库的当前后端协议联调；不使用真实设备/联系人。
4. 给现有 CI/发布聚合门槛增加手表协议检查与 SHA 绑定 artifact；不把该检查称为原生 HAP 构建。
5. 更新仓库说明、完成报告及桌面交接文档，注明导入范围、历史证据、当前结果和真机/传感器限制。
6. 提交并推送至同一分支，等待新 SHA 的 CI/Security，再保存最终 SHA 验收记录。

## 验收边界

这是 HarmonyOS/ArkTS Wearable 模拟联调端；手动生成的心率/步数始终标为 simulated。
未增加真实传感器、后台采集、医疗功能或真机签名。旧工程文档中的 2026-10-25 安装/启动
记录属于历史记录；本次原生构建、协议检查和后端联调必须各自提供新记录。
已知 SOS 服务端没有 healthBatch 同等的 bindingVersion 条件，重绑定时需停止求助操作并重新
认证；此次源码归档不静默改变后端协议。真实数据本地加密和真机验收仍是后续工作。

## 本次执行结果

源码固定于 `0dc3715ef0e8dcc197cbcf59ee84bbfd8a2c5b9b`，从干净检出重新安装锁定依赖并验证：

- 6 项协议测试通过，npm audit 无漏洞；测试可独立于 DevEco 运行。
- DevEco API 24 原生构建成功，生成 unsigned debug HAP；SDK 静态异常警告及未签名警告保留。
- 与同提交未改动的 Go 后端、独立真实 MySQL 8.0.43 联调，6 项检查通过：认证、绑定、心跳、
  healthBatch 入库/去重、SOS 幂等且无真实收件人、解绑重绑后旧批次由客户端及服务端拒绝。
- 生产 ETS、资源、module 配置与原目录内容一致（仅换行不作为代码差异）；本次改动为测试
  可移植性、忽略规则、说明与 CI，未借导入改写手表业务或后端 API。
- 构建缓存、依赖、历史 artifacts、安装包、签名材料和私有配置不进入 Git；原独立目录保持原状。

原始日志及 HAP hash 见 `evidence/watch-import-20261026/`。HAP 不提交到源码仓库。
本次没有重新执行模拟器安装、视觉/完整点击或真机测试，旧 2026-10-25 记录不能替代这些验收。
联调创建独立 `suining_watch_import_*` 测试库，测试服务在结束后停止，原 MySQL Windows 服务不参与。

当前分支的 CI 新增 Watch protocol，并纳入 Release gate / required；发布脚本要求 watch-<SHA>
artifact。该 CI 运行的是 Node/TypeScript 协议检查，不是托管 runner 上的 HarmonyOS SDK 构建。
新文档/证据提交后的 SHA 仍须重新通过本分支 CI/Security，原 30464e5 绿灯不能替代。

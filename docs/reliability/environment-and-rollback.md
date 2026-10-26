# 环境、重现与回滚

## 本次隔离环境

Windows amd64，Go 1.26.10（官方 archive 校验），GCC 15.2.0 race，Node 运行时与 npm 11.6.2；
MySQL 8.0.43，PG 16.15，SeaweedFS 4.51 S3，Prometheus 3.5.0，Alertmanager 0.28.1，
Python TLS 演练依赖 cryptography 50.0.2。真实 CI 另使用 Linux MySQL 8.4/PG 16，Go 1.26.10
和 1.27.2；当前源漏洞扫描 Go 1.27.3。CI artifact 保存实际数据库版本、镜像 ID、模块列表。

源码独立 clone；原解压目录和原 MySQL Windows 服务均未修改。实验 MySQL 仅绑定 loopback。
最初 E 盘 HDD 数据目录 DDL 超过 5 秒超时，因此保留失败，换独立 NVMe 数据目录重跑；不改
事务确认或 fsync。33371 为基准套件，33372 为每次新 UUID 数据目录的进程故障库；55471 为
独立测试 PG，18373 为专用私有 S3。原 Windows 服务不参与任何 stop/drop 操作。
MySQL innodb_flush_log_at_trx_commit=1，PG fsync 保持启用。实际端口/路径见每次 environment.json。
验收完成后已停止本次独立 MySQL/PG/S3 服务，数据与工具保留；原 MySQL2077 Windows 服务保持原状。

凭据、临时 CA/私钥、数据库和 S3 数据位于 `.reliability-lab` 或专用 Temp 数据根，**不进入仓库**。
临时 CA 只给演练 HTTP client，不安装系统信任，不使用 InsecureSkipVerify。所有设备、账号、
号码和接收端均为合成 fixture；不会联系真实家属。工具下载/测试使用本机代理，子服务清除代理
与继承的业务环境变量。数据库密码用环境传给客户端，不写命令行日志。

## 重现入口

CI 可直接在新分支 push/Actions 执行。配置真实 TEST_MYSQL_DSN、TEST_DATABASE_URL、
TEST_S3_ENDPOINT/BUCKET/ACCESS_KEY/SECRET_KEY 后，在 backend 执行：

```text
go test -json -race -count=1 -timeout 600s ./...
go build ./...
go vet ./...
python scripts/generate-openapi.py --check
python -m unittest discover -s scripts/reliability -p test_*.py -v
```

后两条从仓库根执行。PG-only 是单独运行 db/handlers/care/analytics，不用数据库缺省跳过
冒充通过；audit-go-results.py 分别审核 mysql/postgres JSONL。前端从 oldIot 运行 npm ci、
test、typecheck、build、audit，具体命令以 reusable-frontend.yml 为准。
Windows 使用 validate-version.ps1、PowerShell parser 和 native sidecar 构建；不是完整安装器/
真实手表验收。完整 installer 是通过 release gate 后的 preview-candidate 手动入口。

进程矩阵脚本为 local-fault-drill.py，要求干净提交、由该提交构建的 server.exe/reliability.exe、
私有 lab-config 和独立输出目录。当前脚本面向此 Windows 隔离根，主动拒绝其他数据根。
它是真实破坏性测试库演练，不是对任意已存在生产库的通用运维脚本。默认不缩短实际任务
lease/升级 deadline。`--smoke` 只验证 setup，complete=false，不能用于整矩阵验收。

## 回滚

用户仓库只增加此新分支；未合并前可保持 preview 原样，无需重置用户已有工作。
如果后续合并需回退，先维护停写/停发送并备份当前 DB+blob+key；按提交 revert，保留新写数据。
事务/状态机/旧 worker CAS 不得为回退而删除。不要使用 reset --hard/force push 改写共享历史。

数据库回滚脚本在 migrations/rollback；0014 撤销索引，0013 保留目录与墓碑。回滚后需正向
重复迁移测试。不能让旧二进制读取它不支持的 S3/keyring；详情见 storage-operations.md。
移除网关/监控配置时先启用人工备用流程，不能只停 worker 消除告警。密钥错失必须失败关闭，
恢复受托旧密钥或修复迁移，不能自动创建新 key 继续运行。

## 仍待生产证据

真实硬件/渠道送达；独立人员接手和二人复核；异机寻呼与整机失效；生产体量恢复、RPO/RTO、
最旧备份密钥恢复；明文历史迁移；多 ingress 全局限额；管理员应用保护规则与独立 review；
签名 tag、代码签名证书。隔离环境的合成 actor 状态转换不等于真人救援。

# 图片、多实例和恢复操作

`BLOB_BACKEND=local` 使用 IMAGE_DIR；`s3` 使用私有专用 bucket、S3_ENDPOINT、S3_REGION、
S3_BUCKET、S3_ACCESS_KEY_FILE、S3_SECRET_KEY_FILE，配置见 deploy/object-storage.compose.yml。
非 loopback 的 endpoint 必须 HTTPS。对象键限于 suining-images/，bucket 禁止公共访问；
测试中的匿名 GET 必须拒绝。S3 需要共享 MySQL 与 DATA_ENCRYPTION_KEY，全部副本配置一致。
接口仍执行上传大小、图片解码/尺寸和重新编码检查，最多 5 MiB 编码图片，不信任文件扩展名。

对象 AEAD scope 绑定文件键，目录保存明文内容 SHA-256 与大小；下载验证后才返回图像，设置
private/no-store。上传先创建不可变对象再建立目录，DB 明确失败执行补偿；提交结果不明时保留
对象由 GC 处置，避免删除已引用对象。引用写入和 GC 使用同一目录行锁；GC 先提交 deleting
墓碑再物理删除，防止进程崩溃后旧 key 被重新引用。孤儿不会被当作业务成功。

维护工具位于 backend/cmd/reliability，使用受限配置执行；不是公开 HTTP 管理接口：

```text
reliability --action verify-images
reliability --action gc-images --grace 24h
reliability --action gc-images --grace 24h --apply
reliability --action backup-lock --owner <unique-backup-id>
reliability --action backup-unlock --owner <same-backup-id>
```

GC 默认只读，至少 24 小时宽限，跳过仍有引用和保留默认头像。所有应用/维护进程须共用同一
目录库；不得允许 bucket lifecycle 或人工删除绕过引用保护。部署凭据按读/写/维护角色划分，
生产采用加密卷、对象版本/异地备份和访问审计。SHA-256 用于完整性，不代替 AEAD 身份认证。

## 协调备份

1. 停止 schema 变更；有旧版本/非受控图片写入者时进入维护停写窗口。持唯一 owner 获取
   backup-lock，此时 GC 被阻止。备份租约 1 小时；长任务在到期前以同 owner 重新获取来续期。
   续期失败、租约过期、任何对象缺失或 hash 失败均使备份无效，不能宣布完成。
2. 用 MySQL InnoDB 事务一致 dump（单事务、TCP、UTF-8、触发器等按实例需要），同时保存
   schema_migrations 与引用清单。不备份 PG 充当业务数据。备份操作本身也要最小权限凭据。
3. 持有租约期间复制私有不可变对象全集或快照引用集，保存对象键、密文 SHA-256、大小、
   数据库 dump hash、snapshot 时间、源码 SHA、配置/密钥版本。因为对象不可变且 GC 被禁，
   快照后的多余对象允许保留；快照引用的对象必须全部存在。不要用 T1/T2 随意复制宣称原子性。
4. 对备份包独立加密、异地留存。恢复密钥单独托管，绝不放入同包。完成离线核对后释放租约。

## 隔离恢复验收

恢复到全新 MySQL 与私有对象空间，禁用真实通知，只接合成网关。导入 SQL、迁回对象、注入
对应长期 key/lookupKey，运行 verify-images 核对四处引用与内容 hash，然后启动应用；逐项
核对照护事件/审计、outbox、会话、密钥和图片。新主启用前停掉旧发送者并核对网关幂等。
记录实测 RPO/RTO，不能把这次小型合成 fixture 的时延用作生产容量承诺。

TestImageS3TwoInstancesAndRealRestore 实际运行两个 HTTP 实例、独立数据库会话、真实 S3 服务，
A 上传/B 读取/A 下线/B 继续读；做真实 mysqldump 后删除专用测试库和测试对象，再还原，核对
引用/hash、SOS、待发 outbox 和 session。原始失败和后续修复日志都保留。测试不会把临时私钥、
session 或原始 dump 放入证据仓库。

## 旧图片迁移与回滚

切换本地→S3 前需要受控停写/备份，列出所有四处引用（含旧头像），读取本地旧明文/密文并
验证格式，以同逻辑或新 key 加密写入私有对象，回填目录 hash 并运行全引用检查。只改环境变量
会丢失旧本地图片可见性；当前没有自动复制历史目录的迁移命令，必须单独实施并验收。
内置 kai.jpg 仅作为缺省头像后备，不掩盖其他图片缺失。

MySQL 0013 rollback 保留目录/墓碑和历史图像；0014 可撤销新增索引。旧二进制不支持 S3/新
keyring，回滚前必须离线转回其支持的本地图片和 key 格式，或者部署仍能读取新格式的修复版本。
不得删除目录、密钥或已写业务记录来“恢复兼容”。禁止对在线唯一副本执行实验中的 drop/清空。

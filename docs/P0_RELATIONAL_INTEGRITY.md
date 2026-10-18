# 当前设备归属完整性

Go migrations/0004_device_oldperson_fk.sql 建立 device.old_id 到 oldperson.old_id 的外键并使用 RESTRICT 删除。绑定事务锁定长者行；删除遇到外键竞争返回 409。后台注册锁定长者并条件更新 family_id，避免不同邀请码覆盖归属。

已有孤儿记录时迁移失败并回滚该版本，不删除设备或历史。检查查询和人工处置说明在 backend/README.md。真实 PostgreSQL 竞争测试位于 integration_test.go；模拟浏览器测试不能替代它。

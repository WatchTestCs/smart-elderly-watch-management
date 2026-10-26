#!/usr/bin/env python3
"""Generate a field inventory from sanitized information_schema metadata, never rows.

Input TSV: table, column, type, comma-separated actual index names, extra.
Classification is a conservative engineering design, not a claim of compliance or
implemented blanket encryption. Review changes whenever the schema changes.
"""
import argparse
import csv
from pathlib import Path


def classify(table, column, indexes):
    field = table+"."+column
    level, storage = "S2 内部业务/关联元数据", "明文结构化；磁盘/备份另行保护"
    query = "按关系、状态或时间查询；不加密关系 ID、调度字段"
    scheme, migration = "保留必要结构；最小权限、审计与整盘加密", "无需字段格式变更；按保留期处理历史副本"
    encrypted = {
        "web_session.payload": "session",
        "auth_factor.secret": "totp:<role>:<actor>（见 factorScope）",
        "care_notification_outbox.recipient": "care-notification-recipient",
        "message_outbox.recipient": "notification-recipient",
        "care_incident.resolution_note": "care-text",
        "care_incident_event.note": "care-text",
        "alert_review.reason": "care-text",
    }
    if field in encrypted or (table in {"device_location", "alarm_precess", "alarm_process"} and column in {"lng", "lat"}) or (table == "device" and column in {"last_lng", "last_lat"}):
        scope = encrypted.get(field, "location")
        level, storage = "S4 高敏感", "配置 DATA_ENCRYPTION_KEY 时新写 AEAD；历史明文可能仍存在"
        query, scheme = "解密后展示；过滤使用关系 ID/时间/独立摘要", "AES-256-GCM；scope="+scope+"；认证 key ID；独立 lookupKey"
        migration = "保留 A 读取，B 写入；分批重封装并核对；旧明文与备份单独盘点；禁止静默新建密钥"
    elif field == "rule_state.evidence":
        level, storage = "S4 健康证据", "JSON 内 evidence 密文（scope=rule-evidence）；历史格式需盘点"
        query, scheme, migration = "外层状态参与规则查询，证据按需解密", "仅证据值 AEAD；状态/关系继续结构化", "保留 JSON 结构，逐行校验重封装，禁止把密文当证据"
    elif field in {"users.password", "family.family_password"}:
        level, storage = "S4 认证", "bcrypt 单向密码哈希；历史弱格式需审计"
        query, scheme, migration = "认证校验；不解密、不按密码索引", "继续慢哈希；不改为可逆加密", "成功认证后升级旧哈希；弱/未知历史值要求重置"
    elif field in {"device.secret", "web_session.id_hash", "auth_factor.recovery_hashes", "care_notification_outbox.recipient_key", "login_audit.account_hash", "login_audit.ip_hash"}:
        level, storage = "S4 凭据/关联摘要", "单向 SHA-256 或 scoped HMAC；非匿名数据"
        query, scheme = "精确匹配/去重/认证；索引见实际索引列", "高熵令牌哈希；低熵手机号/IP 用独立密钥 HMAC；不可直接 SHA 手机号"
        migration = "加密 key 与 lookupKey 分离；旧 HMAC/明文 SHA 去重键需停写重建或双读迁移；不可直接换 lookupKey"
    elif (column == "code" and table in {"invite_code", "bind_code"}) or (column in {"delivery_token", "lease_token"}):
        level, storage = "S4 短期凭据/租约", "当前明文；须限时、单次、最小访问"
        query, scheme, migration = "精确令牌匹配/工作 CAS", "用户兑换码设计为随机高熵+哈希查找；CAS 租约保留不透明值并禁止日志输出", "扩列双写摘要、双读限期；旧码过期后移除明文；CAS 变更需回归"
    elif table in {"health_sample", "health_change", "metric_daily", "ratepre", "watch_info", "watch_day_info", "watch_day_info_archive"}:
        level, storage = "S4 健康/可关联样本", "明文（含 JSON、派生值、归档）"
        query, scheme = "时间范围、owner、来源、统计聚合；生成列/索引依赖明文", "阶段一保留热查询结构并加密卷/备份；阶段二按授权投影或受控解密重设计 AEAD；不可直接加密 JSON 后破坏统计"
        migration = "新版本 schema/投影双跑核对；重建日统计/索引；覆盖 change、归档、删除标记和备份；当前未实施字段加密"
    elif table in {"device_fence", "fence_state", "fence_event", "analytics_outbox", "rule_policy"}:
        level, storage = "S4 位置/健康/规则载荷", "明文 JSON/数值/元数据；不是全库密文"
        query, scheme = "围栏计算/投影/规则匹配需读取；关系索引保留", "敏感 payload AEAD + 最小必要授权计算；隔离分析投影和磁盘密钥"
        migration = "扩展格式版本，双读校验，生产者/消费者同步切换；旧事件、重放队列和 PG 投影一并处理；当前仅设计"
    elif table in {"oldperson", "family", "customer", "users", "said", "text", "nice_count"} and column not in {"id", "old_id", "family_id", "customer_id", "user_id", "said_id", "text_id", "pic_id", "add_id", "nice_count"}:
        level, storage = "S3 人员/联系/自由文本（可能含 S4 健康）", "当前明文；图片列为不透明引用"
        query = "展示/精确手机号登录/部分列表搜索；不得假设所有字段无需查询"
        scheme = "姓名/地址/病史/自由文本 AEAD；手机号规范化后 scoped HMAC lookup + AEAD；图片值只存引用"
        migration = "扩列密文和 blind index；双写/回填/校验唯一性；查询改造后切读；清除旧列与历史副本；当前未实施"
    elif table in {"device", "device_location", "alarm_precess", "alarm_process"}:
        level = "S3 可关联设备/位置元数据"
        scheme = "保留必需路由、状态、时间；身份绑定受控；坐标列见 AEAD 分类"
    if field in {"family.family_tou_pic", "said.tou_pic", "pic.pic_name", "pic_show.pic_name"}:
        level, storage = "S3 私有图片引用", "明文不透明文件键；不授予公开访问"
        query, scheme, migration = "数据库关联/GC 引用检测", "对象本体 AEAD + SHA-256；DB 引用锁和私有 bucket", "本地旧图片需复制/加密/核对后切换；不可只改 BLOB_BACKEND"
    return [field, level, storage, query, indexes or "无当前索引", scheme, migration]


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("schema_tsv")
    parser.add_argument("output_csv")
    args = parser.parse_args()
    rows = list(csv.reader(Path(args.schema_tsv).read_text(encoding="utf-8-sig").splitlines(), delimiter="\t"))
    if not rows or any(len(row) != 5 for row in rows):
        raise ValueError("expected five metadata columns and at least one field")
    with Path(args.output_csv).open("w", newline="", encoding="utf-8-sig") as out:
        writer = csv.writer(out)
        writer.writerow(["字段", "敏感级别", "当前存储", "查询需求", "当前实际索引", "保护方案（含设计项）", "迁移方案/边界"])
        for table, column, kind, indexes, extra in rows:
            writer.writerow(classify(table, column, indexes))
    print(f"Inventoried {len(rows)} fields across {len(set(row[0] for row in rows))} tables; design is not blanket encryption")

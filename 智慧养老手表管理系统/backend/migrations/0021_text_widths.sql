-- 放宽文本列宽：以下列宽不足以容纳真实录入内容，例如 room varchar(5) 装不下
-- 「A栋301室」（6 字符），在 STRICT_TRANS_TABLES 下会直接 1406 报错，最终表现为
-- 500「服务处理失败」。放宽后配合 handler 层的 vstrMax rune 校验，让超长输入
-- 得到明确的 400 而不是服务端错误。
--
-- 只做扩大，不缩小、不改类型，历史数据不受影响。
ALTER TABLE oldperson ALTER COLUMN name TYPE varchar(50);
ALTER TABLE oldperson ALTER COLUMN room TYPE varchar(50);
ALTER TABLE oldperson ALTER COLUMN illness TYPE varchar(200);
ALTER TABLE oldperson ALTER COLUMN warn TYPE varchar(50);
ALTER TABLE oldperson ALTER COLUMN food_like TYPE varchar(100);
ALTER TABLE oldperson ALTER COLUMN food_hate TYPE varchar(100);
ALTER TABLE oldperson ALTER COLUMN drag TYPE varchar(100);
ALTER TABLE family ALTER COLUMN family_name TYPE varchar(50);
ALTER TABLE family ALTER COLUMN family_address TYPE varchar(200);
ALTER TABLE customer ALTER COLUMN customer_name TYPE varchar(50);
ALTER TABLE customer ALTER COLUMN customer_address TYPE varchar(200);

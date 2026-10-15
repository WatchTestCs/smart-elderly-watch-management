-- Embedded schema version 1. No account or personal seed data. Idempotent on restart.
CREATE TABLE IF NOT EXISTS `alarm_precess` (
  `id` int NOT NULL auto_increment,
  `status` varchar(20) default NULL,
  `old_id` int default NULL,
  `device_id` int default NULL,
  `be_time` datetime default NULL,
  `lng` varchar(20) default NULL,
  `lat` varchar(20) default NULL,
  PRIMARY KEY  (`id`)
);

CREATE TABLE IF NOT EXISTS `customer` (
  `customer_id` int NOT NULL auto_increment,
  `customer_name` varchar(20) default NULL,
  `customer_age` int default NULL,
  `customer_sex` char(1) default NULL,
  `customer_phone` varchar(12) default NULL,
  `customer_address` varchar(20) default NULL,
  PRIMARY KEY  (`customer_id`)
);

CREATE TABLE IF NOT EXISTS `family` (
  `family_tou_pic` varchar(50) default 'kai.jpg',
  `family_id` int NOT NULL auto_increment,
  `family_name` varchar(10) default NULL,
  `family_relation` varchar(10) default NULL,
  `family_phone` varchar(12) default NULL,
  `family_sex` char(1) default NULL,
  `family_address` varchar(20) default NULL,
  `family_password` varchar(60) default NULL,
  PRIMARY KEY  (`family_id`),
  UNIQUE KEY `uk_family_phone` (`family_phone`)
);

CREATE TABLE IF NOT EXISTS `invite_code` (
  `id` int NOT NULL auto_increment,
  `code` varchar(20) NOT NULL,
  `old_id` int default NULL,
  `create_user` varchar(20) default '管理员',
  `create_time` datetime default NULL,
  `expire_time` datetime default NULL,
  `used` tinyint NOT NULL default '0',
  `used_time` datetime default NULL,
  `used_family_id` int default NULL,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uk_invite_code_code` (`code`),
  KEY `idx_invite_code_old` (`old_id`)
);

CREATE TABLE IF NOT EXISTS `nice_count` (
  `phone` varchar(30) default NULL,
  `said_id` int default NULL,
  `add_id` int NOT NULL auto_increment,
  PRIMARY KEY  (`add_id`),
  KEY `idx_nice_count_said_phone` (`said_id`, `phone`)
);

CREATE TABLE IF NOT EXISTS `oldperson` (
  `old_id` int NOT NULL auto_increment,
  `name` varchar(10) default NULL,
  `age` int default NULL,
  `sex` char(1) default NULL,
  `phone` varchar(12) default NULL,
  `room` varchar(5) default NULL,
  `illness` varchar(20) default NULL,
  `warn` varchar(5) default NULL,
  `food_like` varchar(20) default NULL,
  `food_hate` varchar(20) default NULL,
  `drag` varchar(20) default NULL,
  `family_id` int default NULL,
  `customer_id` int default NULL,
  `be_time` date default NULL,
  `bothdate` date default NULL,
  `device_id` int default NULL,
  PRIMARY KEY  (`old_id`)
);

CREATE TABLE IF NOT EXISTS `pic` (
  `pic_id` int NOT NULL,
  `pic_name` varchar(50) default NULL
);

CREATE TABLE IF NOT EXISTS `pic_show` (
  `id` int NOT NULL auto_increment,
  `pic_name` varchar(50) default NULL,
  PRIMARY KEY  (`id`)
);

CREATE TABLE IF NOT EXISTS `ratepre` (
  `id` int NOT NULL auto_increment,
  `device_id` int default NULL,
  `old_id` int default NULL,
  `rate_state` varchar(5) default NULL,
  `pre_state` varchar(5) default NULL,
  `create_time` datetime default NULL,
  `hr` int default NULL,
  `bp` int default NULL,
  PRIMARY KEY  (`id`)
);

CREATE TABLE IF NOT EXISTS `said` (
  `said_id` int NOT NULL auto_increment,
  `said` varchar(2000) default NULL,
  `text_id` int default NULL,
  `back` varchar(2000) default NULL,
  `said_time` datetime default NULL,
  `back_time` datetime default NULL,
  `person_name` varchar(10) default NULL,
  `said_phone` varchar(30) default NULL,
  `tou_pic` varchar(50) default NULL,
  `nice_count` int default '0',
  PRIMARY KEY  (`said_id`)
);

CREATE TABLE IF NOT EXISTS `text` (
  `text_id` int NOT NULL auto_increment,
  `title` varchar(200) default NULL,
  `text_main` varchar(10000) default NULL,
  `time` datetime default NULL,
  `pic_id` int default NULL,
  PRIMARY KEY  (`text_id`)
);

CREATE TABLE IF NOT EXISTS `users` (
  `user_id` int NOT NULL auto_increment,
  `user_name` varchar(10) default NULL,
  `password` varchar(60) default NULL,
  `role` varchar(10) default NULL,
  `user_phone` varchar(20) default NULL,
  PRIMARY KEY  (`user_id`),
  UNIQUE KEY `uk_users_phone` (`user_phone`)
);

CREATE TABLE IF NOT EXISTS `watch_day_info` (
  `id` int NOT NULL auto_increment,
  `device_id` int default NULL,
  `sleep` int default NULL,
  `step` int default NULL,
  `sleep_sate` char(1) default NULL,
  `create_time` date default NULL,
  PRIMARY KEY  (`id`)
);

CREATE TABLE IF NOT EXISTS `watch_info` (
  `device_id` int NOT NULL auto_increment,
  `op_time` date default NULL,
  `bpf` int default NULL,
  `bpl` int default NULL,
  `hrf` int default NULL,
  `hrl` int default NULL,
  PRIMARY KEY  (`device_id`)
);

CREATE TABLE IF NOT EXISTS `device` (
  `device_id` int NOT NULL auto_increment,
  `device_no` varchar(32) NOT NULL,
  `secret` varchar(64) NOT NULL,
  `alias` varchar(32) default NULL,
  `model` varchar(32) default NULL,
  `app_version` varchar(20) default NULL,
  `battery` int default NULL,
  `old_id` int default NULL,
  `family_id` int default NULL,
  `status` tinyint NOT NULL default '1',
  `bind_time` datetime default NULL,
  `last_time` datetime default NULL,
  `last_lng` varchar(20) default NULL,
  `last_lat` varchar(20) default NULL,
  `create_time` datetime default NULL,
  PRIMARY KEY  (`device_id`),
  UNIQUE KEY `uk_device_no` (`device_no`),
  KEY `idx_device_old` (`old_id`)
);

CREATE TABLE IF NOT EXISTS `device_location` (
  `id` int NOT NULL auto_increment,
  `device_id` int default NULL,
  `old_id` int default NULL,
  `lng` varchar(20) default NULL,
  `lat` varchar(20) default NULL,
  `source` varchar(20) default NULL,
  `be_time` datetime default NULL,
  `create_time` datetime default NULL,
  PRIMARY KEY  (`id`),
  KEY `idx_loc_old_time` (`old_id`, `id`),
  KEY `idx_loc_device_time` (`device_id`, `id`)
);

CREATE TABLE IF NOT EXISTS `bind_code` (
  `id` int NOT NULL auto_increment,
  `code` varchar(20) NOT NULL,
  `old_id` int default NULL,
  `family_id` int default NULL,
  `create_user` varchar(20) default NULL,
  `create_time` datetime default NULL,
  `expire_time` datetime default NULL,
  `used` tinyint NOT NULL default '0',
  `used_time` datetime default NULL,
  `used_device_id` int default NULL,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uk_bind_code` (`code`),
  KEY `idx_bind_code_old` (`old_id`)
);

ALTER TABLE ratepre ADD COLUMN IF NOT EXISTS rate_handled int default 0 not null;

ALTER TABLE ratepre ADD COLUMN IF NOT EXISTS pre_handled int default 0 not null;

ALTER TABLE ratepre ADD COLUMN IF NOT EXISTS rate_handled_at timestamp;

ALTER TABLE ratepre ADD COLUMN IF NOT EXISTS pre_handled_at timestamp;

ALTER TABLE device ADD COLUMN IF NOT EXISTS person_handled int default 0 not null;

ALTER TABLE device ADD COLUMN IF NOT EXISTS rate_handled int default 0 not null;

ALTER TABLE device ADD COLUMN IF NOT EXISTS pre_handled int default 0 not null;

ALTER TABLE device ADD COLUMN IF NOT EXISTS person_handled_at timestamp;

ALTER TABLE device ADD COLUMN IF NOT EXISTS boot_seq int default 0 not null;

ALTER TABLE device ADD COLUMN IF NOT EXISTS handled_seq int default 0 not null;

ALTER TABLE device ADD COLUMN IF NOT EXISTS last_online_at timestamp;

CREATE UNIQUE INDEX IF NOT EXISTS uk_day_device_date ON watch_day_info(device_id,create_time);

CREATE INDEX IF NOT EXISTS idx_ratepre_old_time ON ratepre(old_id,create_time,id);

CREATE INDEX IF NOT EXISTS idx_ratepre_device_time ON ratepre(device_id,create_time,id);

CREATE INDEX IF NOT EXISTS idx_alarm_old_time ON alarm_precess(old_id,be_time,id);

CREATE INDEX IF NOT EXISTS idx_old_family ON oldperson(family_id);

CREATE INDEX IF NOT EXISTS idx_said_text ON said(text_id,said_time);

CREATE INDEX IF NOT EXISTS idx_pic_group ON pic(pic_id);

CREATE SEQUENCE IF NOT EXISTS picture_group START WITH 1;

CREATE TABLE IF NOT EXISTS app_metadata (id int primary key, schema_version int not null);

INSERT INTO app_metadata(id,schema_version) SELECT 1,1 WHERE NOT EXISTS (SELECT 1 FROM app_metadata WHERE id=1);

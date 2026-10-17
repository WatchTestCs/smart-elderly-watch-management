package com.example.careold;

import com.example.careold.config.DatabaseMigrations;
import com.example.careold.common.DeviceSecretCodec;
import com.zaxxer.hikari.HikariDataSource;
import java.nio.file.Path;
import java.nio.file.Files;
import org.flywaydb.core.Flyway;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.core.io.ClassPathResource;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.init.ResourceDatabasePopulator;
import static org.junit.jupiter.api.Assertions.*;

class DatabaseMigrationTests {
    @TempDir Path temporary;

    private Flyway flyway(HikariDataSource source) {
        return Flyway.configure().dataSource(source).baselineOnMigrate(true).baselineVersion("0").load();
    }
    private HikariDataSource source(String name) {
        var source = new HikariDataSource();
        source.setJdbcUrl("jdbc:h2:file:" + temporary.resolve(name).toString().replace('\\','/')
                + ";MODE=MySQL;DATABASE_TO_LOWER=TRUE");
        source.setUsername("sa"); source.setMaximumPoolSize(4);
        return source;
    }
    @Test void freshInstallAndRepeatedStartupApplyEachVersionOnce() {
        try (var source = source("fresh")) {
            var flyway = flyway(source);
            DatabaseMigrations.migrate(flyway);
            var jdbc = new JdbcTemplate(source);
            jdbc.update("insert into users(user_name,password) values('验收','preserved')");
            DatabaseMigrations.migrate(flyway);
            assertEquals(4,jdbc.queryForObject("select schema_version from app_metadata where id=1",Integer.class));
            assertEquals(4,flyway.info().applied().length);
            assertEquals("preserved",jdbc.queryForObject("select password from users",String.class));
        }
    }
    @Test void legacyDuplicatesAreArchivedAndBackupCanBeRestored() throws Exception {
        try (var source = source("legacy")) {
            new ResourceDatabasePopulator(new ClassPathResource("db/migration/V1__initial_schema.sql")).execute(source);
            var jdbc = new JdbcTemplate(source);
            jdbc.update("insert into watch_day_info(device_id,step,create_time) values(1,100,'2026-01-01'),(1,200,'2026-01-01'),(2,300,'2026-01-01')");
            String oldSecret = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ";
            jdbc.update("insert into device(device_no,secret,alias) values(?,?,?)","W-OLD",oldSecret,"测试设备");
            DatabaseMigrations.migrate(flyway(source));
            assertEquals(2,jdbc.queryForObject("select count(*) from watch_day_info",Integer.class));
            assertEquals(100,jdbc.queryForObject("select step from watch_day_info_archive",Integer.class));
            assertEquals(200,jdbc.queryForObject("select step from watch_day_info where device_id=1",Integer.class));
            String stored = jdbc.queryForObject("select secret from device where device_no='W-OLD'",String.class);
            assertTrue(DeviceSecretCodec.isHashed(stored));
            assertNotEquals(oldSecret, stored);
            assertTrue(DeviceSecretCodec.matches(oldSecret, stored));
            assertFalse(DeviceSecretCodec.matches("WRONG-23456789ABCDEFGHJKLMNPQ", stored));
            assertEquals(4,jdbc.queryForObject("select schema_version from app_metadata where id=1",Integer.class));
            DatabaseMigrations.migrate(flyway(source));
            assertEquals(stored,jdbc.queryForObject("select secret from device where device_no='W-OLD'",String.class));
            assertThrows(org.springframework.dao.DataIntegrityViolationException.class, () ->
                    jdbc.update("insert into watch_day_info(device_id,create_time) values(1,'2026-01-01')"));
            Path backup;
            try (var backups = Files.list(temporary.resolve("backups"))) { backup = backups.findFirst().orElseThrow(); }
            Path restored = temporary.resolve("restored");
            Files.createDirectories(restored);
            org.h2.tools.Restore.execute(backup.toString(),restored.toString(),"legacy");
            try (var restoredSource = source("restored/legacy")) {
                var restoredJdbc = new JdbcTemplate(restoredSource);
                assertEquals(3,restoredJdbc.queryForObject("select count(*) from watch_day_info",Integer.class));
                assertEquals("23456789ABCDEFGHJKLMNPQRSTUVWXYZ",
                        restoredJdbc.queryForObject("select secret from device where device_no='W-OLD'",String.class));
                assertEquals(1,restoredJdbc.queryForObject("select schema_version from app_metadata",Integer.class));
            }
        }
    }
    @Test void futureDatabaseIsRejectedBeforeAnyMigrationOrBackup() {
        try (var source = source("future")) {
            var jdbc = new JdbcTemplate(source);
            jdbc.execute("create table app_metadata(id int primary key,schema_version int)");
            jdbc.update("insert into app_metadata values(1,99)");
            assertThrows(IllegalStateException.class, () -> DatabaseMigrations.migrate(flyway(source)));
            assertEquals(99,jdbc.queryForObject("select schema_version from app_metadata",Integer.class));
            assertFalse(Files.exists(temporary.resolve("backups")));
            assertEquals(1,jdbc.queryForObject("select count(*) from information_schema.tables where table_schema='public'",Integer.class));
        }
    }

    @Test void validBindingIsEnforcedAndDeletionRequiresExplicitUnbind() {
        try (var source = source("references")) {
            DatabaseMigrations.migrate(flyway(source));
            var jdbc = new JdbcTemplate(source);
            jdbc.update("insert into oldperson(old_id,name) values(1,'保留的长者')");
            jdbc.update("insert into device(device_id,device_no,secret,old_id) values(1,?,?,1)",
                    "TEST-REL", DeviceSecretCodec.hash("a".repeat(32)));
            assertThrows(org.springframework.dao.DataIntegrityViolationException.class, () ->
                    jdbc.update("insert into device(device_no,secret,old_id) values(?,?,999)",
                            "ORPHAN",DeviceSecretCodec.hash("b".repeat(32))));
            assertThrows(org.springframework.dao.DataIntegrityViolationException.class, () ->
                    jdbc.update("update device set old_id=999 where device_id=1"));
            assertThrows(org.springframework.dao.DataIntegrityViolationException.class, () ->
                    jdbc.update("delete from oldperson where old_id=1"));
            assertEquals(1,jdbc.queryForObject("select old_id from device where device_id=1",Integer.class));
            jdbc.update("update device set old_id=null where device_id=1");
            assertEquals(1,jdbc.update("delete from oldperson where old_id=1"));
            assertEquals(4,jdbc.queryForObject("select schema_version from app_metadata where id=1",Integer.class));
        }
    }

    @Test void orphanedLegacyReferenceFailsClosedWithBackupBeforeConstraint() throws Exception {
        try (var source = source("v3-orphan")) {
            var oldFlyway = Flyway.configure().dataSource(source)
                    .baselineOnMigrate(true).baselineVersion("0").target("3").load();
            DatabaseMigrations.migrate(oldFlyway);
            var jdbc = new JdbcTemplate(source);
            jdbc.update("insert into device(device_id,device_no,secret,old_id) values(1,?,?,404)",
                    "BAD-REF", DeviceSecretCodec.hash("a".repeat(32)));
            assertThrows(org.flywaydb.core.api.FlywayException.class, () ->
                    DatabaseMigrations.migrate(flyway(source)));
            assertEquals(404,jdbc.queryForObject("select old_id from device where device_id=1",Integer.class));
            assertEquals(3,jdbc.queryForObject("select schema_version from app_metadata where id=1",Integer.class));
            Path backup;
            try (var copies=Files.list(temporary.resolve("backups"))) {
                backup=copies.findFirst().orElseThrow();
            }
            Path restored = temporary.resolve("restored-reference");
            Files.createDirectories(restored);
            org.h2.tools.Restore.execute(backup.toString(),restored.toString(),"v3-orphan");
            try(var restoredSource=source("restored-reference/v3-orphan")) {
                var restoredJdbc = new JdbcTemplate(restoredSource);
                assertEquals(404,restoredJdbc.queryForObject("select old_id from device where device_id=1",Integer.class));
                assertEquals(3,restoredJdbc.queryForObject("select schema_version from app_metadata where id=1",Integer.class));
            }
        }
    }

}

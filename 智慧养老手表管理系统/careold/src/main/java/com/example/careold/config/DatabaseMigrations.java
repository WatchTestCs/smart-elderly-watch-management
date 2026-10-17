package com.example.careold.config;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.sql.SQLException;
import org.flywaydb.core.Flyway;
import org.flywaydb.core.api.MigrationVersion;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.autoconfigure.flyway.FlywayMigrationStrategy;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class DatabaseMigrations {
    private static final Logger LOG = LoggerFactory.getLogger(DatabaseMigrations.class);
    private static final int SCHEMA_VERSION = 2;

    @Bean
    FlywayMigrationStrategy guardedMigrations() { return DatabaseMigrations::migrate; }

    public static void migrate(Flyway flyway) {
        var information = flyway.info();
        try (var connection = flyway.getConfiguration().getDataSource().getConnection();
             var statement = connection.createStatement()) {
            var tables = connection.getMetaData().getTables(null, null, "app_metadata", new String[]{"TABLE"});
            boolean hasMetadata;
            try (tables) { hasMetadata = tables.next(); }
            if (hasMetadata) {
                try (var version = statement.executeQuery("select schema_version from app_metadata where id=1")) {
                    if (version.next() && version.getInt(1) > SCHEMA_VERSION) {
                        throw new IllegalStateException("数据库来自更新版本，拒绝降级打开；请使用对应或更新的岁宁版本");
                    }
                }
            }
            for (var migration : information.applied()) {
                if (migration.getVersion() != null && migration.getVersion().compareTo(MigrationVersion.fromVersion("2")) > 0) {
                    throw new IllegalStateException("数据库迁移版本高于当前软件，拒绝降级打开");
                }
            }
            boolean hasUserTables;
            try (var count = statement.executeQuery("select count(*) from information_schema.tables where table_schema='public' and table_name<>'flyway_schema_history'")) {
                count.next(); hasUserTables=count.getInt(1)>0;
            }
            if (hasUserTables && information.pending().length > 0) {
                String databasePath;
                try (var result = statement.executeQuery("select DATABASE_PATH()")) {
                    result.next(); databasePath = result.getString(1);
                }
                if (databasePath != null) {
                    Path backupDir = Path.of(databasePath).getParent().resolve("backups");
                    Files.createDirectories(backupDir);
                    Path backup = backupDir.resolve("before-v2-" + java.util.UUID.randomUUID() + ".zip");
                    statement.execute("BACKUP TO '" + backup.toString().replace("'", "''") + "'");
                    LOG.info("数据库升级前备份已完成：{}", backup);
                }
            }
        } catch (SQLException | IOException e) {
            throw new IllegalStateException("数据库升级检查或备份失败；为保护原始数据已停止启动", e);
        }
        var result = flyway.migrate();
        if (result.migrationsExecuted > 0) LOG.info("数据库迁移完成：{} 项，版本 {}", result.migrationsExecuted, result.targetSchemaVersion);
    }
}

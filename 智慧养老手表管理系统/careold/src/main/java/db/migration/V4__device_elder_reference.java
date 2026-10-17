package db.migration;

import java.sql.SQLException;
import org.flywaydb.core.api.migration.BaseJavaMigration;
import org.flywaydb.core.api.migration.Context;

/**
 * The first integrity constraint in a staged relationship redesign.
 * Device.old_id is an active binding; elder removal requires an explicit
 * unbind. Historical orphan IDs block the upgrade, never get nulled/deleted.
 */
public class V4__device_elder_reference extends BaseJavaMigration {
    @Override
    public void migrate(Context context) throws Exception {
        var connection = context.getConnection();
        try (var statement = connection.prepareStatement(
                "select d.device_id, d.old_id from device d " +
                "left join oldperson o on o.old_id=d.old_id " +
                "where d.old_id is not null and o.old_id is null " +
                "order by d.device_id limit 1");
             var invalid=statement.executeQuery()) {
            if (invalid.next()) {
                throw new SQLException("Cannot enforce device.old_id reference: " +
                        "orphan device_id=" + invalid.getInt(1) +
                        " refers to missing old_id=" + invalid.getInt(2) +
                        "; restore the pre-upgrade backup and repair deliberately");
            }
        }
        try (var statement = connection.createStatement()) {
            statement.execute("ALTER TABLE device ADD CONSTRAINT fk_device_oldperson " +
                    "FOREIGN KEY (old_id) REFERENCES oldperson(old_id) ON DELETE RESTRICT");
            if (statement.executeUpdate("UPDATE app_metadata SET schema_version=4 WHERE id=1") != 1) {
                throw new SQLException("Missing schema metadata during device relationship upgrade");
            }
        }
    }
}

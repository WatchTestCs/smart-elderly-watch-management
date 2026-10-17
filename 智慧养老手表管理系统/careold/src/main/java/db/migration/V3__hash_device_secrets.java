package db.migration;

import com.example.careold.common.DeviceSecretCodec;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.flywaydb.core.api.migration.BaseJavaMigration;
import org.flywaydb.core.api.migration.Context;

/**
 * V3: one-way conversion of pre-existing device secrets without changing the
 * device JSON contract. The guarded migration runner makes a pre-upgrade H2
 * backup; restore that backup to roll back to the V2 application.
 */
public class V3__hash_device_secrets extends BaseJavaMigration {
    @Override
    public void migrate(Context context) throws Exception {
        var connection = context.getConnection();
        List<Map.Entry<Integer,String>> changes = new ArrayList<>();
        try (var statement = connection.prepareStatement("select device_id, secret from device");
             var rows = statement.executeQuery()) {
            while (rows.next()) {
                int id = rows.getInt(1);
                String stored = rows.getString(2);
                if (DeviceSecretCodec.isHashed(stored)) continue;
                if (stored == null || stored.startsWith("sha256$")) {
                    throw new SQLException("Invalid credential representation on device " + id);
                }
                // Fail closed on corrupted legacy credentials rather than quietly
                // changing credentials that the old application could not accept.
                changes.add(Map.entry(id, DeviceSecretCodec.hash(stored)));
            }
        }
        try (var update = connection.prepareStatement("update device set secret=? where device_id=?")) {
            for (var change : changes) {
                update.setString(1, change.getValue());
                update.setInt(2, change.getKey());
                if (update.executeUpdate() != 1) throw new SQLException("Device credential migration lost a row");
            }
        }
        try (var metadata = connection.prepareStatement("update app_metadata set schema_version=3 where id=1")) {
            if (metadata.executeUpdate() != 1) throw new SQLException("Missing app_metadata row");
        }
    }
}

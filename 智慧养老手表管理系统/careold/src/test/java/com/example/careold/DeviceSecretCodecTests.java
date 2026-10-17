package com.example.careold;

import com.example.careold.common.DeviceSecretCodec;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class DeviceSecretCodecTests {
    private static final String CLEAR = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ";

    @Test void saltedDomainSeparatedDigestFitsExistingDatabaseColumn() {
        var encoded = DeviceSecretCodec.hash(CLEAR);
        assertTrue(DeviceSecretCodec.isHashed(encoded));
        assertEquals(50, encoded.length());
        assertFalse(encoded.contains(CLEAR));
        assertTrue(DeviceSecretCodec.matches(CLEAR, encoded));
        assertFalse(DeviceSecretCodec.matches("WRONG-23456789ABCDEFGHJKLMNPQ", encoded));
        assertFalse(DeviceSecretCodec.matches(CLEAR, CLEAR)); // no post-upgrade plaintext fallback
    }

    @Test void malformedOrShortSecretsAreNeverAuthenticated() {
        assertFalse(DeviceSecretCodec.matches("", DeviceSecretCodec.hash(CLEAR)));
        assertFalse(DeviceSecretCodec.matches(CLEAR, "sha256$not-a-complete-hash"));
        assertFalse(DeviceSecretCodec.matches(CLEAR, null));
        assertThrows(IllegalArgumentException.class, () -> DeviceSecretCodec.hash("demo-secret-1"));
    }
}

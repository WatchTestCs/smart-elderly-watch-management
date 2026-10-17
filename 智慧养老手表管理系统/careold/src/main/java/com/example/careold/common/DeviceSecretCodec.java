package com.example.careold.common;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Base64;

/**
 * Hashing for server-generated, high-entropy device credentials. Unlike human
 * passwords, these credentials are 32 characters from a SecureRandom alphabet;
 * a domain-separated SHA-256 digest is safe from practical offline guessing.
 * The wire credential is unchanged, but the database never needs to store it.
 */
public final class DeviceSecretCodec {
    private static final String PREFIX = "sha256$";
    private static final byte[] DOMAIN = "suining-device-secret-v1\0".getBytes(StandardCharsets.UTF_8);

    private DeviceSecretCodec() {}

    public static boolean isHashed(String stored) {
        return stored != null && stored.matches("sha256\\$[A-Za-z0-9_-]{43}");
    }

    public static String hash(String clear) {
        if (clear == null || clear.length() < 24 || clear.length() > 64) {
            throw new IllegalArgumentException("Invalid device credential length");
        }
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            digest.update(DOMAIN);
            byte[] bytes = digest.digest(clear.getBytes(StandardCharsets.UTF_8));
            return PREFIX + Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
        } catch (NoSuchAlgorithmException impossible) {
            throw new IllegalStateException("SHA-256 is unavailable", impossible);
        }
    }

    public static boolean matches(String supplied, String stored) {
        if (!isHashed(stored) || supplied == null || supplied.length() < 24 || supplied.length() > 64) return false;
        String expected = hash(supplied);
        return MessageDigest.isEqual(expected.getBytes(StandardCharsets.US_ASCII),
                stored.getBytes(StandardCharsets.US_ASCII));
    }
}

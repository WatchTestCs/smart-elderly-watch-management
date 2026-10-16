package com.example.careold.common;

import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;

public final class PasswordUtil {

    private static final BCryptPasswordEncoder ENCODER = new BCryptPasswordEncoder(12);

    private PasswordUtil() {
    }

    public static String hash(String password) {
        if (password == null || password.getBytes(java.nio.charset.StandardCharsets.UTF_8).length > 72) {
            throw new IllegalArgumentException("密码不能超过72个UTF-8字节");
        }
        return ENCODER.encode(password);
    }

    public static boolean validNewPassword(String password) {
        return password != null && password.matches("(?s)(?=.*[A-Za-z])(?=.*[0-9]).{12,20}")
                && password.getBytes(java.nio.charset.StandardCharsets.UTF_8).length <= 72;
    }

    public static boolean matches(String password, String storedPassword) {
        if (password == null || storedPassword == null || password.getBytes(java.nio.charset.StandardCharsets.UTF_8).length > 72) {
            return false;
        }
        if (storedPassword.startsWith("$2")) {
            try {
                return isHashed(storedPassword) && ENCODER.matches(password, storedPassword);
            } catch (IllegalArgumentException e) {
                return false;
            }
        }
        return false;
    }

    public static boolean isHashed(String password) {
        return password != null && password.matches("\\$2[aby]\\$\\d{2}\\$[./A-Za-z0-9]{53}");
    }
}

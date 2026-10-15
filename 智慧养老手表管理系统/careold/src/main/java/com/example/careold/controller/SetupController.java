package com.example.careold.controller;

import com.example.careold.common.PasswordUtil;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Map;

/** Desktop-only, one-time setup. The random token stays in the native process. */
@RestController
@RequestMapping("/restful/setup")
public class SetupController {
    private final JdbcTemplate jdbc;
    private final String token;
    public SetupController(JdbcTemplate jdbc, @Value("${SUINING_DESKTOP_TOKEN:}") String token) {
        this.jdbc = jdbc;
        this.token = token;
    }

    @PostMapping
    @Transactional
    public Map<String, Object> create(@RequestBody Map<String, String> input, HttpServletRequest request) {
        String supplied = request.getHeader("X-Suining-Desktop");
        if (token.isBlank() || supplied == null || !MessageDigest.isEqual(token.getBytes(StandardCharsets.UTF_8), supplied.getBytes(StandardCharsets.UTF_8)))
            throw new ResponseStatusException(HttpStatus.FORBIDDEN);
        // Serialize simultaneous setup attempts across all requests.
        jdbc.queryForObject("SELECT id FROM app_metadata WHERE id=1 FOR UPDATE", Integer.class);
        if (jdbc.queryForObject("SELECT COUNT(*) FROM users", Integer.class) != 0)
            throw new ResponseStatusException(HttpStatus.CONFLICT, "管理员已创建");
        String phone = input.getOrDefault("phone", "").trim();
        String name = input.getOrDefault("name", "").trim();
        String password = input.getOrDefault("password", "");
        if (!phone.matches("1\\d{10}") || name.isBlank() || name.length() > 10 || password.length() < 12
                || password.getBytes(StandardCharsets.UTF_8).length > 72 || !password.matches("(?s).*[A-Za-z].*") || !password.matches("(?s).*\\d.*"))
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "请检查姓名、手机号和密码");
        jdbc.update("INSERT INTO users(user_name,user_phone,password,role) VALUES(?,?,?,'1')", name, phone, PasswordUtil.hash(password));
        return Map.of("returnCode", "1111");
    }
}

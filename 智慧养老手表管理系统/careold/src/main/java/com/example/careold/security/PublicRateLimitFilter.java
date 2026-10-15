package com.example.careold.security;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

/** 限制暴力登录和公开邀请码探测；有界存储，不信任客户端伪造的代理 IP。 */
public class PublicRateLimitFilter extends OncePerRequestFilter {
    private record Window(long start, int count) { }
    private final Map<String, Window> windows = new HashMap<>();
    private static final long WINDOW_MS = 10 * 60 * 1000L;

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        String path = request.getServletPath();
        int limit = switch (path) {
            case "/restful/setup", "/restful/userLogin/login", "/restful/inviteCode/join", "/restful/device/bind" -> 30;
            case "/restful/inviteCode/checkCode", "/restful/device/login" -> 60;
            default -> 0;
        };
        if (limit > 0 && !allow(request.getRemoteAddr() + ':' + path, limit, System.currentTimeMillis())) {
            response.setStatus(429);
            response.setHeader("Retry-After", "600");
            response.setContentType("application/json;charset=UTF-8");
            response.getWriter().write("{\"returnCode\":\"0000\",\"message\":\"尝试次数较多，请稍后再试\"}");
            return;
        }
        chain.doFilter(request, response);
    }

    synchronized boolean allow(String key, int limit, long now) {
        Window window = windows.get(key);
        if (window == null || now - window.start() >= WINDOW_MS) {
            if (windows.size() >= 10000) windows.entrySet().removeIf(entry -> now - entry.getValue().start() >= WINDOW_MS);
            if (!windows.containsKey(key) && windows.size() >= 10000) return false;
            windows.put(key, new Window(now, 1));
            return true;
        }
        if (window.count() >= limit) return false;
        windows.put(key, new Window(window.start(), window.count() + 1));
        return true;
    }
}

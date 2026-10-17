package com.example.careold.security;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpMethod;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.csrf.CsrfFilter;
import org.springframework.security.web.csrf.CsrfTokenRepository;
import org.springframework.security.web.csrf.HttpSessionCsrfTokenRepository;
import org.springframework.security.web.context.HttpSessionSecurityContextRepository;
import jakarta.servlet.http.HttpServletResponse;

@Configuration
public class SecurityConfig {
    private static final String[] HARDWARE = {
            "/restful/device/login", "/restful/device/bind", "/restful/device/heartbeat",
            "/restful/device/report", "/restful/device/sos"
    };
    private static final String[] SHARED = {
            "/restful/family/selectTwo", "/restful/family/updateTwo", "/restful/family/updateTouPic", "/restful/family/updatePassword",
            "/restful/ratepre/selectRatepreFamily", "/restful/ratepre/selectRatepreDetail",
            "/restful/step/selectStepFamily", "/restful/step/selectStepDetail", "/restful/position/selectFamily",
            "/restful/text/select", "/restful/text/selectDetail", "/restful/picshow/select",
            "/restful/said/select", "/restful/said/insert", "/restful/said/goodCount", "/restful/image/saveImage"
    };

    @Bean
    HttpSessionSecurityContextRepository securityContextRepository() {
        return new HttpSessionSecurityContextRepository();
    }

    @Bean
    CsrfTokenRepository csrfTokenRepository() {
        return new HttpSessionCsrfTokenRepository();
    }

    @Bean
    SecurityFilterChain securityFilterChain(HttpSecurity http, HttpSessionSecurityContextRepository contexts,
                                          CsrfTokenRepository tokens) throws Exception {
        http.securityContext(context -> context.securityContextRepository(contexts))
                .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.IF_REQUIRED))
                .csrf(csrf -> csrf.csrfTokenRepository(tokens).ignoringRequestMatchers(HARDWARE).ignoringRequestMatchers("/restful/desktop/exit"))
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers(HttpMethod.GET, "/restful/userLogin/session").permitAll()
                        .requestMatchers(HttpMethod.POST, "/restful/desktop/exit", "/restful/setup", "/restful/userLogin/login", "/restful/inviteCode/checkCode", "/restful/inviteCode/join").permitAll()
                        .requestMatchers(HttpMethod.POST, HARDWARE).permitAll()
                        .requestMatchers(HttpMethod.POST, SHARED).hasAnyRole("ADMIN", "FAMILY")
                        .requestMatchers(HttpMethod.GET, "/restful/image/download").hasAnyRole("ADMIN", "FAMILY")
                        .requestMatchers("/restful/**").hasRole("ADMIN")
                        .requestMatchers("/", "/index.html", "/assets/**", "/static/**", "/favicon.ico", "/error").permitAll()
                        .anyRequest().denyAll())
                .exceptionHandling(errors -> errors
                        .authenticationEntryPoint((request, response, error) -> jsonError(response, 401, "AUTH_REQUIRED", "请重新登录"))
                        .accessDeniedHandler((request, response, error) -> jsonError(response, 403,
                                error instanceof org.springframework.security.web.csrf.CsrfException ? "CSRF_REJECTED" : "ACCESS_DENIED",
                                "请求未获授权，请刷新页面后重试")))
                .headers(headers -> headers
                        .contentSecurityPolicy(csp -> csp.policyDirectives("default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https://tile.openstreetmap.org; font-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'"))
                        .referrerPolicy(policy -> policy.policy(org.springframework.security.web.header.writers.ReferrerPolicyHeaderWriter.ReferrerPolicy.STRICT_ORIGIN_WHEN_CROSS_ORIGIN)))
                .requestCache(cache -> cache.disable())
                .formLogin(form -> form.disable())
                .httpBasic(basic -> basic.disable())
                .logout(logout -> logout.logoutUrl("/restful/userLogin/logout").deleteCookies("JSESSIONID")
                        .logoutSuccessHandler((request, response, authentication) -> {
                            response.setContentType("application/json;charset=UTF-8");
                            response.getWriter().write("{\"returnCode\":\"1111\"}");
                        }))
                .addFilterBefore(new PublicRateLimitFilter(), CsrfFilter.class);
        return http.build();
    }

    private static void jsonError(HttpServletResponse response, int status, String code, String message) throws java.io.IOException {
        response.setStatus(status);
        response.setContentType("application/json;charset=UTF-8");
        response.getWriter().write("{\"returnCode\":\"0000\",\"code\":\"" + code + "\",\"message\":\"" + message + "\"}");
    }
}

package com.example.careold.controller;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Map;

@RestController
public class DesktopController {
    private final ConfigurableApplicationContext context;
    private final String token;
    public DesktopController(ConfigurableApplicationContext context, @Value("${SUINING_DESKTOP_TOKEN:}") String token) {
        this.context=context; this.token=token;
    }
    @PostMapping("/restful/desktop/exit")
    public Map<String,Boolean> exit(@RequestHeader(value="X-Suining-Desktop",defaultValue="") String supplied) {
        if (token.isBlank() || !MessageDigest.isEqual(token.getBytes(StandardCharsets.UTF_8), supplied.getBytes(StandardCharsets.UTF_8)))
            throw new ResponseStatusException(HttpStatus.FORBIDDEN);
        Thread close = new Thread(() -> {
            try { Thread.sleep(200); } catch (InterruptedException e) { Thread.currentThread().interrupt(); }
            context.close();
        },"desktop-shutdown");
        close.setDaemon(false); close.start();
        return Map.of("closing",true);
    }
}

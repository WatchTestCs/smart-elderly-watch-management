package com.example.careold.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.boot.web.servlet.context.ServletWebServerApplicationContext;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;
import java.io.IOException;
import java.nio.file.*;

/** Publish readiness only after schema initialization and the web server are ready. */
@Component
public class DesktopReady {
    @Value("${SUINING_READY_FILE:}") String readyFile;
    @Value("${SUINING_PARENT_PID:0}") long parentPid;
    @EventListener(ApplicationReadyEvent.class)
    public void ready(ApplicationReadyEvent event) throws IOException {
        if (readyFile.isBlank()) return;
        int port = ((ServletWebServerApplicationContext) event.getApplicationContext()).getWebServer().getPort();
        Path target = Path.of(readyFile);
        Files.createDirectories(target.getParent());
        Path temporary = Files.createTempFile(target.getParent(), "ready-", ".tmp");
        Files.writeString(temporary, "{\"port\":" + port + ",\"pid\":" + ProcessHandle.current().pid() + "}");
        Files.move(temporary, target, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
        if (parentPid > 0) {
            Thread watcher = new Thread(() -> {
                while (ProcessHandle.of(parentPid).map(ProcessHandle::isAlive).orElse(false)) {
                    try { Thread.sleep(2000); } catch (InterruptedException e) { return; }
                }
                event.getApplicationContext().close();
            }, "desktop-parent-watch");
            watcher.setDaemon(true);
            watcher.start();
        }
    }
}

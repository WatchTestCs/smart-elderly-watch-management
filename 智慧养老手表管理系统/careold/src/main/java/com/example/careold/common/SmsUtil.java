package com.example.careold.common;

import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.security.*;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.*;

@Component
public class SmsUtil {
    private static final Logger logger = LoggerFactory.getLogger(SmsUtil.class);
    @Value("${tencent.sms.app-id:}") private String appId;
    @Value("${tencent.sms.app-key:}") private String appKey;
    private final ObjectMapper json;
    private final SecureRandom random = new SecureRandom();
    private final HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(5)).build();
    private final ExecutorService workers = new ThreadPoolExecutor(1, 2, 30, TimeUnit.SECONDS,
            new ArrayBlockingQueue<>(100), task -> { Thread t=new Thread(task,"notice-sms"); t.setDaemon(true); return t; });
    public SmsUtil(ObjectMapper json) { this.json = json; }

    public void sendMessage(String phone) {
        if (appId == null || !appId.matches("\\d+") || appKey == null || appKey.isBlank() || phone == null || !phone.matches("1\\d{10}")) return;
        Runnable enqueue = () -> {
            try { workers.execute(() -> send(phone)); }
            catch (RejectedExecutionException e) { logger.warn("短信队列已满，通知未发送"); }
        };
        // Never hold a database lock during network I/O or notify about a rolled-back notice.
        if (TransactionSynchronizationManager.isSynchronizationActive()) {
            TransactionSynchronizationManager.registerSynchronization(new TransactionSynchronization() {
                @Override public void afterCommit() { enqueue.run(); }
            });
        } else enqueue.run();
    }

    private void send(String phone) {
        try {
            long nonce = random.nextInt(900000) + 100000L, time = System.currentTimeMillis() / 1000;
            String material = "appkey=" + appKey + "&random=" + nonce + "&time=" + time + "&mobile=" + phone;
            String signature = HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(material.getBytes(StandardCharsets.UTF_8)));
            Map<String,Object> body = Map.of("tel",Map.of("nationcode","86","mobile",phone),"type",0,
                    "msg","你关注的养老院有新的公告动态了，请登录系统查看","sig",signature,"time",time,"extend","","ext","");
            var request = HttpRequest.newBuilder(URI.create("https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=" + appId + "&random=" + nonce))
                    .timeout(Duration.ofSeconds(10)).header("Content-Type","application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(json.writeValueAsString(body))).build();
            var response = client.send(request,HttpResponse.BodyHandlers.ofString());
            if (response.statusCode()!=200 || json.readTree(response.body()).path("result").asInt(-1)!=0)
                logger.warn("短信服务未接受公告通知");
        } catch (InterruptedException e) { Thread.currentThread().interrupt(); }
        catch (Exception e) { logger.warn("公告短信发送失败: {}",e.getClass().getSimpleName()); }
    }
    @PreDestroy public void close() { workers.shutdownNow(); }
}

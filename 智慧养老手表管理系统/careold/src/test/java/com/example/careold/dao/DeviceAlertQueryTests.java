package com.example.careold.dao;

import com.example.careold.common.DateConst;
import org.junit.jupiter.api.Test;
import org.springframework.jdbc.core.JdbcTemplate;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class DeviceAlertQueryTests {
    @Test
    public void refreshingTwoHundredDevicesUsesOneQuery() {
        List<Map<String, Object>> devices = new ArrayList<>();
        for (int id = 1; id <= 200; id++) devices.add(device(id));
        RecordingJdbc jdbc = new RecordingJdbc(devices);
        DeviceJdbcDao dao = new DeviceJdbcDao();
        dao.jdbcTemplate = jdbc;
        assertEquals(200, dao.countPendingAlerts().size());
        assertEquals(1, jdbc.queries, "List refresh must not query each device and profile separately");
    }

    @Test
    public void preservesPersonnelReasonsAndPendingCounts() {
        List<Map<String, Object>> devices = new ArrayList<>();
        for (int id = 1; id <= 8; id++) devices.add(device(id));
        devices.get(1).put("oldId", null);
        devices.get(2).put("battery", 19);
        devices.get(3).put("hrf", null);
        devices.get(4).put("lastTime", null); // acknowledged, no new alert
        devices.get(5).put("lastTime", null);
        devices.get(5).put("bootSeq", 2); // a new offline episode
        devices.get(6).put("lastTime", "2000-01-01 00:00:00");
        devices.get(6).put("bootSeq", 2);
        devices.get(7).put("battery", 0); // preserve existing zero-battery semantics
        devices.get(7).put("rateCount", 3L);
        devices.get(7).put("preCount", 2L);
        DeviceJdbcDao dao = new DeviceJdbcDao();
        dao.jdbcTemplate = new RecordingJdbc(devices);
        List<Map<String, Object>> result = dao.countPendingAlerts();
        int[] expected = {0, 1, 1, 1, 0, 1, 1, 0};
        for (int index = 0; index < expected.length; index++) assertEquals(expected[index], result.get(index).get("personCount"));
        assertEquals(3, result.get(7).get("rateCount"));
        assertEquals(2, result.get(7).get("preCount"));
    }

    private static Map<String, Object> device(int id) {
        Map<String, Object> row = new LinkedHashMap<>();
        row.put("deviceId", id); row.put("deviceNo", "CARE-" + id); row.put("alias", "测试设备");
        row.put("oldId", id); row.put("oldPersonName", "测试长者");
        row.put("lastTime", DateConst.dateToString(new Date(), DateConst.DATE_MODEL_2));
        row.put("createTime", "2026-01-01 00:00:00"); row.put("battery", 80);
        row.put("bootSeq", 1); row.put("handledSeq", 1);
        row.put("hrf", 110); row.put("hrl", 55); row.put("bpf", 140); row.put("bpl", 90);
        row.put("rateCount", 0L); row.put("preCount", 0L);
        return row;
    }

    private static class RecordingJdbc extends JdbcTemplate {
        final List<Map<String, Object>> devices;
        int queries;
        RecordingJdbc(List<Map<String, Object>> devices) { this.devices = devices; }
        @Override public List<Map<String, Object>> queryForList(String sql) { queries++; return devices; }
        @Override public List<Map<String, Object>> queryForList(String sql, Object... args) {
            queries++;
            for (Map<String, Object> row : devices) if (row.get("deviceId").equals(args[0])) return Collections.singletonList(row);
            return Collections.emptyList();
        }
    }
}

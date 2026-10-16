package com.example.careold.controller;

import com.example.careold.dao.*;
import com.zaxxer.hikari.HikariDataSource;
import java.util.LinkedHashMap;
import javax.sql.DataSource;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.core.io.ClassPathResource;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.DataSourceTransactionManager;
import org.springframework.jdbc.datasource.init.ResourceDatabasePopulator;
import org.springframework.test.context.junit.jupiter.SpringJUnitConfig;
import org.springframework.transaction.annotation.EnableTransactionManagement;
import org.springframework.web.server.ResponseStatusException;

import static org.junit.jupiter.api.Assertions.*;

/** Real database failures exercise proxy transaction boundaries, without user data or mocks. */
@SpringJUnitConfig(DeviceWriteTransactionTests.Config.class)
class DeviceWriteTransactionTests {
    @Configuration
    @EnableTransactionManagement
    @Import({com.example.careold.service.DeviceService.class, DeviceJdbcDao.class, OldPersonJdbcDao.class, WatchInfoJdbcDao.class,
            InviteCodeController.class, InviteCodeJdbcDao.class, FamilyJdbcDao.class, UserJdbcDao.class,
            RatepreJdbcDao.class, StepJdbcDao.class, PositionJdbcDao.class, WatchInfoController.class, FamilyController.class,
            OldPersonController.class, RatepreController.class, CustomerJdbcDao.class, OldPersonDetailJdbcDao.class,
            TextJdbcDao.class, SaidJdbcDao.class})
    static class Config {
        @Bean(destroyMethod="close") DataSource dataSource() {
            HikariDataSource pool = new HikariDataSource();
            pool.setJdbcUrl("jdbc:h2:mem:device-writes;MODE=MySQL;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1");
            pool.setUsername("sa"); pool.setMaximumPoolSize(4);
            return pool;
        }
        @Bean RecordingJdbc jdbcTemplate(DataSource source) { return new RecordingJdbc(source); }
        @Bean DataSourceTransactionManager transactionManager(DataSource source) { return new DataSourceTransactionManager(source); }
    }
    static class RecordingJdbc extends JdbcTemplate {
        int deviceUpdates;
        java.util.List<String> selects = new java.util.ArrayList<>();
        RecordingJdbc(DataSource source) { super(source); }
        @Override public int update(String sql, Object... args) {
            if (sql.toLowerCase().startsWith("update device set")) deviceUpdates++;
            return super.update(sql, args);
        }
        @Override public <T> java.util.List<T> query(String sql, org.springframework.jdbc.core.RowMapper<T> mapper) {
            selects.add(sql); return super.query(sql,mapper);
        }
        @Override public <T> java.util.List<T> query(String sql, org.springframework.jdbc.core.RowMapper<T> mapper, Object... args) {
            selects.add(sql); return super.query(sql,mapper,args);
        }
    }
    @Autowired RecordingJdbc jdbc;
    @Autowired DataSource source;
    @Autowired com.example.careold.service.DeviceService devices;
    @Autowired InviteCodeController invitations;
    @Autowired RatepreDao health;
    @Autowired StepDao activity;
    @Autowired PositionDao positions;
    @Autowired WatchInfoController profiles;
    @Autowired DeviceJdbcDao deviceDao;
    @Autowired OldPersonDao elders;
    @Autowired FamilyController families;
    @Autowired OldPersonController elderController;
    @Autowired RatepreController healthController;
    @Autowired CustomerDao customers;
    @Autowired OldPersonDetailDao elderDetails;
    @Autowired FamilyDao familyDao;
    @Autowired TextDao notices;
    @Autowired SaidDao comments;
    @Autowired InviteCodeDao inviteDao;
    @Autowired WatchInfoDao watchProfiles;

    @BeforeEach void schema() {
        jdbc.execute("drop all objects");
        com.example.careold.config.DatabaseMigrations.migrate(org.flywaydb.core.Flyway.configure().dataSource(source).load());
        jdbc.update("insert into oldperson(old_id,name,device_id) values(1,'测试长者',1),(2,'另一长者',2)");
        jdbc.update("insert into device(device_id,device_no,secret,old_id,status,battery,app_version) values(1,'TEST-01',?,1,1,50,'1.0')", "a".repeat(32));
        jdbc.deviceUpdates = 0;
    }
    private LinkedHashMap request(Object... fields) {
        LinkedHashMap data = new LinkedHashMap();
        data.put("deviceNo", "TEST-01"); data.put("secret", "a".repeat(32));
        for (int i=0;i<fields.length;i+=2) data.put(fields[i], fields[i+1]);
        return data;
    }
    private int count(String table) { return jdbc.queryForObject("select count(*) from " + table, Integer.class); }

    @Test void laterDeviceWriteFailureRollsBackEntireReport() {
        jdbc.execute("alter table device add constraint fail_version check(app_version <> 'fault')");
        assertThrows(DataIntegrityViolationException.class, () -> devices.report(request("hr",75,"bp",120,
                "step",300,"sleep",8,"lng","120.1","lat","30.1","alarm","fall","appVersion","fault")));
        for (String table : new String[]{"ratepre","watch_day_info","device_location","alarm_precess"}) assertEquals(0,count(table),table);
        assertNull(jdbc.queryForObject("select last_lng from device where device_id=1",String.class));
    }
    @Test void failedSosLocationDoesNotLeaveAnAlarm() {
        jdbc.execute("alter table device_location add constraint fail_sos check(source <> 'sos')");
        assertThrows(DataIntegrityViolationException.class, () -> devices.sos(request("lng","120","lat","30")));
        assertEquals(0,count("alarm_precess"));
    }
    @Test void batchInviteFailureRollsBackEarlierCodes() {
        jdbc.execute("alter table invite_code add constraint fail_second check(id <= 1)");
        assertThrows(DataIntegrityViolationException.class, () -> invitations.create(request("oldId",1,"count",2)));
        assertEquals(0,count("invite_code"));
    }
    @Test void configuredThresholdsDriveStoredAndReturnedStates() {
        jdbc.update("insert into watch_info(device_id,hrf,hrl,bpf,bpl) values(1,80,60,180,150)");
        var result = devices.report(request("hr",90,"bp",120));
        assertEquals("1",result.get("rateState")); assertEquals("1",result.get("preState"));
        assertEquals("1",jdbc.queryForObject("select rate_state from ratepre",String.class));
        var alert = jdbc.queryForMap("select hr,bp from ratepre");
        assertEquals(90,alert.get("hr"));
    }
    @Test void missingProfileAndInvertedLegacyBoundsRemainUsable() {
        assertEquals("0",devices.report(request("hr",75,"bp",120)).get("rateState"));
        jdbc.update("insert into watch_info(device_id,hrf,hrl,bpf,bpl) values(1,55,110,90,140)");
        var result = devices.report(request("hr",75,"bp",120));
        assertEquals("0",result.get("rateState")); assertEquals("0",result.get("preState"));
    }
    @Test void partialDailyUpdatesPreserveOtherValuesAndAcceptExplicitZero() {
        devices.report(request("step",400,"sleep",8,"sleepState","1"));
        devices.report(request("step",0));
        var day=jdbc.queryForMap("select step,sleep,sleep_sate from watch_day_info");
        assertEquals(0,day.get("step")); assertEquals(8,day.get("sleep")); assertEquals("1",day.get("sleep_sate"));
        devices.report(request("sleep",0,"sleepState","0"));
        assertEquals(0,jdbc.queryForObject("select sleep from watch_day_info",Integer.class));
        assertEquals(1,count("watch_day_info"));
    }
    @Test void invalidCoordinatesAndMetricsRejectWithoutWrites() {
        for (LinkedHashMap input : new LinkedHashMap[]{request("hr",75,"lng","181","lat","30"),
                request("lng","NaN","lat","30"),request("lng","120"),request("hr",999),request("battery",-1),
                request("beTime","2026-02-30 12:00:00"),request("sleepState","2"),request("appVersion","x".repeat(21)),
                request("source","x".repeat(21)),request("step","NaN")}) {
            assertEquals(400,assertThrows(ResponseStatusException.class, () -> devices.report(input)).getStatusCode().value());
        }
        assertEquals(0,count("ratepre")); assertEquals(0,count("device_location")); assertEquals(0,jdbc.deviceUpdates);
    }
    @Test void heartbeatPreservesOmittedMetadataAndUpdatesDeviceOnce() {
        assertEquals("1111",devices.heartbeat(request()).get("returnCode"));
        assertEquals(1,jdbc.deviceUpdates);
        assertEquals(50,jdbc.queryForObject("select battery from device where device_id=1",Integer.class));
        assertEquals("1.0",jdbc.queryForObject("select app_version from device where device_id=1",String.class));
        assertNotNull(jdbc.queryForObject("select last_online_at from device where device_id=1",java.sql.Timestamp.class));
    }
    @Test void familyLatestHealthUsesTimestampThenIdAndFiltersAfterLatest() {
        jdbc.update("insert into ratepre(old_id,device_id,hr,bp,create_time) values(1,1,60,100,'2026-01-01'),(1,1,70,110,'2026-01-02'),(1,1,80,120,'2026-01-02'),(2,2,99,130,'2026-01-03')");
        assertEquals(80,health.getRatepreFamily(1,"","").get(0).getHr());
        assertTrue(health.getRatepreFamily(1,"2026-01-01 00:00:00","2026-01-01 23:59:59").isEmpty());
        assertTrue(health.getRatepreFamily(999,"","").isEmpty());
    }
    @Test void familyLatestActivityAndPositionDoNotLeakOtherElders() {
        jdbc.update("insert into watch_day_info(device_id,step,create_time) values(1,100,'2026-01-01'),(1,200,'2026-01-02'),(2,999,'2026-01-03')");
        jdbc.update("insert into device_location(device_id,old_id,lng,lat,be_time) values(1,1,'120','30','2026-01-03'),(1,1,'121','31','2026-01-02'),(2,2,'122','32','2026-01-04')");
        assertEquals(200,activity.getStepSleepFamily(1).get(0).getStep());
        assertEquals("121",positions.getPositionFamily(1).get(0).getLng());
        assertTrue(activity.getStepSleepFamily(999).isEmpty());
        assertTrue(positions.getPositionFamily(999).isEmpty());
    }
    @Test void alertExplanationUsesConfiguredRange() {
        jdbc.update("insert into watch_info(device_id,hrf,hrl,bpf,bpl) values(1,80,60,180,150)");
        devices.report(request("hr",90,"bp",120));
        var rate = deviceDao.findPendingAlert(1,"rate");
        var pressure = deviceDao.findPendingAlert(1,"pre");
        assertEquals("心率偏高",rate.get("reason")); assertEquals("60-80",rate.get("threshold"));
        assertEquals("血压偏低",pressure.get("reason")); assertEquals("150-180",pressure.get("threshold"));
        jdbc.update("update watch_info set hrf=110 where device_id=1");
        assertEquals("历史心率异常",deviceDao.findPendingAlert(1,"rate").get("reason"));
    }
    @Test void equalOnlineTimestampsCountOfflineOnceAndRearmAfterHeartbeat() {
        jdbc.update("update device set last_time='2000-01-01',last_online_at='2000-01-01' where device_id=1");
        deviceDao.scanOfflineDevices(); deviceDao.scanOfflineDevices();
        assertEquals(1,jdbc.queryForObject("select boot_seq from device where device_id=1",Integer.class));
        devices.heartbeat(request()); deviceDao.scanOfflineDevices();
        assertEquals(1,jdbc.queryForObject("select boot_seq from device where device_id=1",Integer.class));
        jdbc.update("update device set last_time='2000-01-01',last_online_at='2000-01-01' where device_id=1");
        deviceDao.scanOfflineDevices();
        assertEquals(2,jdbc.queryForObject("select boot_seq from device where device_id=1",Integer.class));
    }
    @Test void incompleteElderAgeRemainsNullInsteadOfFailingOrBecomingZero() {
        assertNull(elders.getOldPersonById(1).getAge());
        assertEquals("1111",invitations.create(request("oldId",1)).get("returnCode"));
    }
    @Test void familyCreationStillAssociatesTheElderSelectedByPhone() {
        jdbc.update("update oldperson set phone='13000000000' where old_id=1");
        var result = families.add(request("familyName","测试家属","familyRelation","女儿","familySex","女",
                "familyPhone","13100000000","oldPhone","13000000000","familyAddress","测试地址"));
        assertEquals("1111",result.get("returnCode"));
        Integer familyId = jdbc.queryForObject("select family_id from family where family_phone='13100000000'",Integer.class);
        assertEquals(familyId,jdbc.queryForObject("select family_id from oldperson where old_id=1",Integer.class));
        assertNull(jdbc.queryForObject("select family_id from oldperson where old_id=2",Integer.class));
    }
    @Test void failedFamilyAssociationRollsBackTheInsertedFamily() {
        jdbc.update("update oldperson set phone='13000000000' where old_id=1");
        jdbc.execute("alter table oldperson add constraint fail_link check(family_id is null)");
        assertThrows(DataIntegrityViolationException.class, () -> families.add(request(
                "familyName","测试家属","familyRelation","女儿","familySex","女","familyPhone","13100000000",
                "oldPhone","13000000000","familyAddress","测试地址")));
        assertEquals(0,count("family"));
    }
    @Test void failedDeviceProfileRollsBackDeviceAndElderLink() {
        jdbc.execute("alter table watch_info add constraint fail_profile check(device_id < 0)");
        assertThrows(DataIntegrityViolationException.class, () -> devices.create(request("oldId",2,"alias","测试新设备")));
        assertEquals(1,count("device"));
        assertEquals(2,jdbc.queryForObject("select device_id from oldperson where old_id=2",Integer.class));
    }
    @Test void actualH2AlertCountsDoNotQueryEachElder() {
        for (int id=2;id<=200;id++) jdbc.update("insert into device(device_id,device_no,secret,old_id,status) values(?,?,?,1,1)",id,"COUNT-"+id,"b".repeat(32));
        jdbc.selects.clear();
        assertEquals(200,deviceDao.countPendingAlerts().size());
        assertEquals(1,jdbc.selects.size(),jdbc.selects.toString());
    }
    @Test void failedElderDeletionRestoresRemovedInvitations() {
        jdbc.update("insert into invite_code(code,old_id) values('ROLLBACK',1)");
        jdbc.execute("create table elder_guard(old_id int references oldperson(old_id))");
        jdbc.update("insert into elder_guard values(1)");
        assertThrows(DataIntegrityViolationException.class, () -> elderController.delete(request("oldId",1)));
        assertEquals(1,count("invite_code"));
        assertEquals(2,count("oldperson"));
    }
    @Test void failedUnbindRestoresReverseElderLink() {
        jdbc.execute("alter table device add constraint fail_unbind check(old_id is not null)");
        assertThrows(DataIntegrityViolationException.class, () -> devices.unbind(request("deviceId",1)));
        assertEquals(1,jdbc.queryForObject("select device_id from oldperson where old_id=1",Integer.class));
    }
    @Test void failedDeviceDeleteRestoresProfileAndReverseLink() {
        jdbc.update("insert into watch_info(device_id,hrf) values(1,110)");
        jdbc.execute("create table device_guard(device_id int references device(device_id))");
        jdbc.update("insert into device_guard values(1)");
        assertThrows(DataIntegrityViolationException.class, () -> devices.delete(request("deviceId",1)));
        assertEquals(1,count("watch_info"));
        assertEquals(1,jdbc.queryForObject("select device_id from oldperson where old_id=1",Integer.class));
    }
    @Test void failedFamilyDeleteRestoresElderAssociation() {
        jdbc.update("insert into family(family_id,family_name) values(1,'家属')");
        jdbc.update("update oldperson set family_id=1 where old_id=1");
        jdbc.execute("create table family_guard(family_id int references family(family_id))");
        jdbc.update("insert into family_guard values(1)");
        assertThrows(DataIntegrityViolationException.class, () -> families.deleteFamily(request("familyId",1)));
        assertEquals(1,jdbc.queryForObject("select family_id from oldperson where old_id=1",Integer.class));
    }
    @Test void dualHealthDispositionRollsBackIfTheSecondWriteFails() {
        jdbc.update("insert into ratepre(id,device_id,old_id,rate_state,pre_state) values(1,1,1,'1','1')");
        jdbc.execute("alter table ratepre add constraint fail_pressure check(pre_handled=0)");
        assertThrows(DataIntegrityViolationException.class, () -> healthController.resolveAlert(request("recordId",1)));
        assertEquals(0,jdbc.queryForObject("select rate_handled from ratepre where id=1",Integer.class));
    }
    @Test void devicePaginationFiltersBeforeLimitingAndDoesNotWrite() {
        for (int id=2;id<=25;id++) jdbc.update("insert into device(device_id,device_no,secret,alias,status) values(?,?,?,'分页设备',1)",id,"PAGE-"+id,"b".repeat(32));
        jdbc.update("update device set last_time='2000-01-01',last_online_at='2000-01-01' where device_id=1");
        int before=jdbc.deviceUpdates;
        var first=devices.list(request("page",1,"pageSize",10,"query","PAGE-","state","offline"));
        var second=devices.list(request("page",2,"pageSize",10,"query","PAGE-","state","offline"));
        assertEquals(24L,((Number)first.get("total")).longValue());
        assertEquals(10,((java.util.List<?>)first.get("devices")).size());
        assertNotEquals(((java.util.List<?>)first.get("devices")).get(0),((java.util.List<?>)second.get("devices")).get(0));
        assertEquals(before,jdbc.deviceUpdates);
        assertEquals(0,jdbc.queryForObject("select boot_seq from device where device_id=1",Integer.class));
        assertThrows(ResponseStatusException.class, () -> devices.list(request("pageSize",1000)));
    }
    @Test void healthActivityAndPositionPaginationUseDatabaseTotalsAndStablePages() {
        for (int id=3;id<=25;id++) jdbc.update("insert into oldperson(old_id,name,device_id) values(?,'分页长者',?)",id,id);
        for (int id=1;id<=25;id++) {
            jdbc.update("insert into ratepre(old_id,device_id,hr,create_time) values(?,?,75,'2026-01-01')",id,id);
            jdbc.update("insert into watch_day_info(device_id,step,create_time) values(?,100,'2026-01-01')",id);
        }
        var page=new com.example.careold.domain.PageRequest(2,10);
        assertEquals(25,health.getRatepre("","","",page).total());
        assertEquals(11,health.getRatepre("","","",page).items().get(0).getOldId());
        assertEquals(10,activity.getStepSleep("","","",page).items().size());
        assertEquals(25,positions.getPosition("",page).total());
        assertEquals(0,health.getRatepre("分页","2027-01-01","2027-12-31",page).total());
        assertThrows(ResponseStatusException.class, () -> new com.example.careold.domain.PageRequest(0,10));
    }
    @Test void allAdministrativeTablesPageThroughRealH2AndFilterInvitesBeforeLimiting() {
        jdbc.update("insert into family(family_id,family_name,family_phone) values(1,'测试家属','13100000000')");
        jdbc.update("insert into customer(customer_id,customer_name,customer_age) values(1,'测试监护',40)");
        for (int id=3;id<=25;id++) jdbc.update("insert into oldperson(old_id,name,device_id) values(?,'分页长者',?)",id,id);
        jdbc.update("update oldperson set family_id=1,customer_id=1");
        for (int id=1;id<=25;id++) {
            jdbc.update("insert into watch_info(device_id,hrf,hrl,bpf,bpl) values(?,110,55,140,90)",id);
            jdbc.update("insert into text(text_id,title,time) values(?,'分页公告',CURRENT_TIMESTAMP)",id);
            jdbc.update("insert into said(said_id,text_id,said,said_time) values(?,?,'分页评论',CURRENT_TIMESTAMP)",id,id);
            jdbc.update("insert into invite_code(code,old_id,used,expire_time) values(?,1,?,?)","CODE-"+id,id%2,id%2==0?java.sql.Timestamp.valueOf("2000-01-01 00:00:00"):null);
        }
        var page=new com.example.careold.domain.PageRequest(2,10);
        assertEquals(25,elders.getOldPersonPage("",page).total());
        assertEquals(10,elderDetails.getOldPersonDetail("",page).items().size());
        assertEquals(25,familyDao.findFamily("测试",page).total());
        assertEquals(1,customers.getCustomer("监护",page).total());
        assertEquals(10,watchProfiles.getWatchInfo("",page).items().size());
        assertEquals(25,notices.selectTextAll("公告",page).total());
        assertEquals(10,comments.getSaidForAdmin(page).items().size());
        assertEquals(12,inviteDao.findInviteCodes("expired",new com.example.careold.domain.PageRequest(1,10)).total());
        assertEquals(13,inviteDao.findInviteCodes("used",page).total());
        assertEquals(0,inviteDao.findInviteCodes("unused",page).total());
    }
    @Test void invalidProfileCannotChangeStoredThresholds() {
        jdbc.update("insert into watch_info(device_id,hrf,hrl,bpf,bpl) values(1,110,55,140,90)");
        assertEquals(400,assertThrows(ResponseStatusException.class, () -> profiles.update(
                request("deviceId",1,"hrf",55,"hrl",110,"bpf",140,"bpl",90))).getStatusCode().value());
        assertEquals(400,assertThrows(ResponseStatusException.class, () -> profiles.update(
                request("deviceId",1,"hrf",999,"hrl",55,"bpf",140,"bpl",90))).getStatusCode().value());
        assertEquals(110,jdbc.queryForObject("select hrf from watch_info where device_id=1",Integer.class));
        assertEquals("1111",profiles.update(request("deviceId",1,"hrf",80,"hrl",60,"bpf",180,"bpl",150)).get("returnCode"));
        assertEquals("1",devices.report(request("hr",90)).get("rateState"));
    }
}

package com.example.careold.controller;

import com.example.careold.dao.*;
import com.zaxxer.hikari.HikariDataSource;
import java.util.LinkedHashMap;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.DataSourceTransactionManager;
import org.springframework.test.context.junit.jupiter.SpringJUnitConfig;
import org.springframework.transaction.annotation.EnableTransactionManagement;
import javax.sql.DataSource;
import java.util.concurrent.*;

import static org.junit.jupiter.api.Assertions.*;

/** 事务和并发测试使用独立内存数据库，不接触用户数据。 */
@SpringJUnitConfig(RegistrationTransactionTests.Config.class)
class RegistrationTransactionTests {
    @Configuration
    @EnableTransactionManagement
    @Import({InviteCodeController.class, InviteCodeJdbcDao.class, FamilyJdbcDao.class, OldPersonJdbcDao.class, UserJdbcDao.class})
    static class Config {
        @Bean(destroyMethod="close") DataSource dataSource() {
            HikariDataSource pool = new HikariDataSource();
            pool.setJdbcUrl("jdbc:h2:mem:transactions;MODE=MySQL;DB_CLOSE_DELAY=-1;LOCK_TIMEOUT=10000");
            pool.setUsername("sa"); pool.setPassword(""); pool.setMaximumPoolSize(4);
            return pool;
        }
        @Bean JdbcTemplate jdbcTemplate(DataSource source) { return new JdbcTemplate(source); }
        @Bean DataSourceTransactionManager transactionManager(DataSource source) { return new DataSourceTransactionManager(source); }
    }
    @Autowired JdbcTemplate jdbc;
    @Autowired InviteCodeController invitations;
    @Autowired InviteCodeDao invitationDao;

    @BeforeEach
    void schema() {
        jdbc.execute("create table if not exists family(family_id int primary key auto_increment,family_name varchar(10),family_relation varchar(10),family_phone varchar(12) unique,family_sex varchar(2),family_address varchar(20),family_password varchar(60)) engine=InnoDB");
        jdbc.execute("create table if not exists users(user_id int primary key,user_name varchar(20),password varchar(60),role varchar(20),user_phone varchar(20) unique) engine=InnoDB");
        jdbc.execute("create table if not exists oldperson(old_id int primary key,name varchar(20),family_id int) engine=InnoDB");
        jdbc.execute("create table if not exists invite_code(id int primary key auto_increment,code varchar(20) unique,old_id int,create_user varchar(20),create_time datetime,expire_time datetime,used int,used_time datetime,used_family_id int) engine=InnoDB");
        for (String table : new String[]{"invite_code","family","oldperson","users"}) jdbc.update("delete from " + table);
        jdbc.update("insert into oldperson(old_id,name) values(1,'测试长辈')");
        jdbc.update("insert into invite_code(code,old_id,used) values('TESTCODE',1,0)");
    }
    private LinkedHashMap registration(String phone) {
        LinkedHashMap request = new LinkedHashMap();
        request.put("code","TESTCODE"); request.put("familyPhone",phone); request.put("familyPassword","Good-secret12345");
        request.put("familyName","测试家属"); request.put("familyRelation","子女"); request.put("familySex","男"); request.put("familyAddress","");
        return request;
    }
    @Test
    void happyPathHashesPasswordAndCommitsAssociationAndCodeTogether() {
        assertNotNull(invitationDao.findByCodeForUpdate("TESTCODE").getOldId());
        assertEquals("1111",invitations.join(registration("13000000000")).get("returnCode"));
        assertTrue(jdbc.queryForObject("select family_password from family",String.class).startsWith("$2a$"));
        assertEquals(jdbc.queryForObject("select family_id from family",Integer.class),jdbc.queryForObject("select family_id from oldperson where old_id=1",Integer.class));
        assertEquals(1,jdbc.queryForObject("select used from invite_code",Integer.class));
    }
    @Test
    void failureAfterAccountInsertRollsBackAccountAssociationAndConsumption() {
        // 在最后一次数据库写入制造真实约束失败，确认早先的两次写入一起回滚。
        jdbc.execute("alter table invite_code add constraint test_consume_failure check (used=0)");
        try {
            assertThrows(RuntimeException.class, () -> invitations.join(registration("13000000000")));
            assertEquals(0,jdbc.queryForObject("select count(*) from family",Integer.class));
            assertNull(jdbc.queryForObject("select family_id from oldperson where old_id=1",Integer.class));
            assertEquals(0,jdbc.queryForObject("select used from invite_code",Integer.class));
        } finally { jdbc.execute("alter table invite_code drop constraint test_consume_failure"); }
    }
    @Test
    void sameInvitationConcurrentUseCreatesOnlyOneAccountAndOneAssociation() throws Exception {
        ExecutorService workers = Executors.newFixedThreadPool(2);
        CountDownLatch ready = new CountDownLatch(2), start = new CountDownLatch(1);
        try {
            Callable<String> first = () -> { ready.countDown(); start.await(); return String.valueOf(invitations.join(registration("13000000000")).get("returnCode")); };
            Callable<String> second = () -> { ready.countDown(); start.await(); return String.valueOf(invitations.join(registration("13100000000")).get("returnCode")); };
            Future<String> a=workers.submit(first),b=workers.submit(second);
            assertTrue(ready.await(5,TimeUnit.SECONDS)); start.countDown();
            int successes=(a.get(15,TimeUnit.SECONDS).equals("1111")?1:0)+(b.get(15,TimeUnit.SECONDS).equals("1111")?1:0);
            assertEquals(1,successes);
            assertEquals(1,jdbc.queryForObject("select count(*) from family",Integer.class));
            assertEquals(jdbc.queryForObject("select family_id from family",Integer.class),jdbc.queryForObject("select family_id from oldperson where old_id=1",Integer.class));
            assertEquals(1,jdbc.queryForObject("select used from invite_code",Integer.class));
        } finally { start.countDown(); workers.shutdownNow(); }
    }

    @Test
    void differentInvitationsCannotCreateDuplicatePhoneAccounts() throws Exception {
        jdbc.update("insert into oldperson(old_id,name) values(2,'另一位测试长辈')");
        jdbc.update("insert into invite_code(code,old_id,used) values('OTHERCODE',2,0)");
        ExecutorService workers = Executors.newFixedThreadPool(2);
        CountDownLatch ready = new CountDownLatch(2), start = new CountDownLatch(1);
        try {
            Callable<String> useFirst = () -> { ready.countDown(); start.await(); return String.valueOf(invitations.join(registration("13000000000")).get("returnCode")); };
            Callable<String> useSecond = () -> {
                ready.countDown(); start.await();
                LinkedHashMap input=registration("13000000000"); input.put("code","OTHERCODE");
                try { return String.valueOf(invitations.join(input).get("returnCode")); }
                catch (org.springframework.dao.DuplicateKeyException expected) { return "conflict"; }
            };
            Future<String> a=workers.submit(useFirst), b=workers.submit(useSecond);
            assertTrue(ready.await(5,TimeUnit.SECONDS)); start.countDown();
            String first;
            try { first=a.get(15,TimeUnit.SECONDS); }
            catch (ExecutionException error) {
                if (!(error.getCause() instanceof org.springframework.dao.DuplicateKeyException)) throw error;
                first="conflict";
            }
            String second=b.get(15,TimeUnit.SECONDS);
            assertEquals(1,(first.equals("1111")?1:0)+(second.equals("1111")?1:0));
            assertEquals(1,jdbc.queryForObject("select count(*) from family",Integer.class));
            assertEquals(1,jdbc.queryForObject("select count(*) from oldperson where family_id is not null",Integer.class));
            assertEquals(1,jdbc.queryForObject("select count(*) from invite_code where used=1",Integer.class));
        } finally { start.countDown(); workers.shutdownNow(); }
    }
}

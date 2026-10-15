package com.example.careold.controller;

import com.example.careold.dao.*;
import com.example.careold.security.AccountPrincipal;
import java.util.LinkedHashMap;
import org.junit.jupiter.api.*;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Import;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.authority.AuthorityUtils;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.test.context.junit.jupiter.SpringJUnitConfig;
import java.util.concurrent.*;
import static org.junit.jupiter.api.Assertions.*;

@SpringJUnitConfig(CommentTransactionTests.Config.class)
class CommentTransactionTests {
    @Configuration
    @Import({RegistrationTransactionTests.Config.class, SaidController.class, SaidJdbcDao.class, NiceCountJdbcDao.class})
    static class Config { }
    @Autowired JdbcTemplate jdbc;
    @Autowired SaidController comments;
    @Autowired SaidDao saidDao;

    private void identity() {
        SecurityContextHolder.getContext().setAuthentication(UsernamePasswordAuthenticationToken.authenticated(
                new AccountPrincipal("FAMILY",7,"13000000000",1,"测试家属"),null,AuthorityUtils.createAuthorityList("ROLE_FAMILY")));
    }
    private LinkedHashMap toggle() {
        LinkedHashMap input=new LinkedHashMap(); input.put("saidId",1); input.put("phone","13900000000");
        return input;
    }
    @BeforeEach
    void schema() {
        jdbc.execute("create table if not exists said(said_id int primary key,said varchar(50),text_id int,nice_count int default 0) engine=InnoDB");
        jdbc.execute("create table if not exists nice_count(add_id int primary key auto_increment,said_id int,phone varchar(30),index idx_nice_count_said_phone(said_id,phone)) engine=InnoDB");
        jdbc.update("delete from nice_count"); jdbc.update("delete from said");
        jdbc.update("insert into said(said_id,said,text_id,nice_count) values(1,'测试评论',1,0)"); identity();
    }
    @AfterEach void clear() { SecurityContextHolder.clearContext(); }
    @Test
    void likeAndUnlikeUseServerIdentityAndReturnSuccess() {
        assertEquals("1111",comments.goodCount(toggle()).get("returnCode"));
        assertEquals("13000000000",jdbc.queryForObject("select phone from nice_count",String.class));
        assertEquals("1",saidDao.getSaidWithLikes(1,"13000000000").get(0).getGood());
        assertEquals("0",saidDao.getSaidWithLikes(1,"13900000000").get(0).getGood());
        assertEquals("1111",comments.goodCount(toggle()).get("returnCode"));
        assertEquals(0,jdbc.queryForObject("select count(*) from nice_count",Integer.class));
        assertEquals(0,jdbc.queryForObject("select nice_count from said where said_id=1",Integer.class));
    }
    @Test
    void counterFailureRollsBackLikeRow() {
        jdbc.execute("alter table said add constraint test_count_failure check(nice_count=0)");
        try {
            assertThrows(RuntimeException.class,()->comments.goodCount(toggle()));
            assertEquals(0,jdbc.queryForObject("select count(*) from nice_count",Integer.class));
        } finally { jdbc.execute("alter table said drop constraint test_count_failure"); }
    }
    @Test
    void concurrentTogglesCannotCreateDuplicateLikesOrLoseCount() throws Exception {
        ExecutorService workers=Executors.newFixedThreadPool(2);
        CountDownLatch ready=new CountDownLatch(2),start=new CountDownLatch(1);
        Callable<Object> action=()->{identity();ready.countDown();start.await();try{return comments.goodCount(toggle()).get("returnCode");}finally{SecurityContextHolder.clearContext();}};
        try {
            Future<Object>a=workers.submit(action),b=workers.submit(action);
            assertTrue(ready.await(5,TimeUnit.SECONDS));start.countDown();
            assertEquals("1111",a.get(15,TimeUnit.SECONDS));assertEquals("1111",b.get(15,TimeUnit.SECONDS));
            assertEquals(0,jdbc.queryForObject("select count(*) from nice_count",Integer.class));
            assertEquals(0,jdbc.queryForObject("select nice_count from said where said_id=1",Integer.class));
        } finally {start.countDown();workers.shutdownNow();}
    }
}

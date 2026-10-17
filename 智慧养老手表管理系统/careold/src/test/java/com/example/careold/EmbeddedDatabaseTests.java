package com.example.careold;

import com.example.careold.dao.*;
import com.example.careold.common.PasswordUtil;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.mock.web.MockHttpSession;
import org.springframework.test.web.servlet.MockMvc;
import static org.junit.jupiter.api.Assertions.*;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest(properties={
    "spring.datasource.url=jdbc:h2:mem:embedded_acceptance;MODE=MySQL;DATABASE_TO_LOWER=TRUE;DB_CLOSE_DELAY=-1",
    "SUINING_DESKTOP_TOKEN=acceptance-only-desktop-token"
})
@AutoConfigureMockMvc
class EmbeddedDatabaseTests {
    @Autowired JdbcTemplate jdbc;
    @Autowired MockMvc mvc;
    @Autowired PositionDao positions;
    @Autowired RatepreDao rates;
    @Autowired StepDao steps;
    @Autowired DeviceJdbcDao devices;
    @Autowired TextDao notices;

    @Test void freshInstallSetupQueriesAndHardwareWritesWorkTogether() throws Exception {
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM users", Integer.class));
        assertEquals(0, jdbc.queryForObject("SELECT COUNT(*) FROM oldperson", Integer.class));
        String input="{\"name\":\"测试管理员\",\"phone\":\"13000000000\",\"password\":\"Acceptance12345\"}";
        mvc.perform(post("/restful/setup").with(csrf()).contentType("application/json").content(input)).andExpect(status().isForbidden());
        mvc.perform(post("/restful/setup").header("X-Suining-Desktop","acceptance-only-desktop-token").contentType("application/json").content(input)).andExpect(status().isForbidden());
        mvc.perform(post("/restful/setup").with(csrf()).header("X-Suining-Desktop","acceptance-only-desktop-token").contentType("application/json").content(input)).andExpect(status().isOk());
        mvc.perform(post("/restful/setup").with(csrf()).header("X-Suining-Desktop","acceptance-only-desktop-token").contentType("application/json").content(input)).andExpect(status().isConflict());
        assertTrue(PasswordUtil.matches("Acceptance12345",jdbc.queryForObject("select password from users",String.class)));
        assertEquals(0, positions.getPosition(null,new com.example.careold.domain.PageRequest(1,20)).items().size());
        assertEquals(0, rates.getRatepre(null,null,null,new com.example.careold.domain.PageRequest(1,20)).items().size());
        assertEquals(0, steps.getStepSleep(null,null,null,new com.example.careold.domain.PageRequest(1,20)).items().size());
        assertTrue(notices.nextPicId() < notices.nextPicId());
        jdbc.update("insert into oldperson(old_id,name,sex,be_time,device_id) values(1,'测试长辈','男',CURRENT_DATE,1)");
        jdbc.update("insert into device(device_id,device_no,secret,old_id,status,create_time) values(1,'ACCEPTANCE','12345678901234567890123456789012',1,1,CURRENT_TIMESTAMP)");
        jdbc.update("insert into device_location(device_id,old_id,lng,lat,be_time) values(1,1,'120.1','30.1',CURRENT_TIMESTAMP)");
        jdbc.update("insert into ratepre(device_id,old_id,hr,bp,rate_state,pre_state,create_time) values(1,1,78,110,'0','0',CURRENT_TIMESTAMP)");
        jdbc.update("insert into watch_day_info(device_id,step,sleep,sleep_sate,create_time) values(1,3000,8,'0',CURRENT_DATE)");
        jdbc.update("insert into watch_day_info(device_id,step,create_time) values(1,3200,CURRENT_DATE) on duplicate key update step=values(step)");
        assertEquals("120.1",positions.getPositionFamily(1).get(0).getLng());
        assertEquals(1,rates.getRatepreFamily(1,null,null).size());
        assertEquals(1,steps.getStepSleepFamily(1).size());
        devices.scanOfflineDevices();
        var login=mvc.perform(post("/restful/userLogin/login").with(csrf()).contentType("application/json").content(input)).andExpect(status().isOk()).andReturn();
        var session=(MockHttpSession)login.getRequest().getSession(false);
        mvc.perform(get("/restful/userLogin/session").session(session)).andExpect(jsonPath("$.authenticated").value(true)).andExpect(jsonPath("$.setupRequired").value(false));
        // A fresh database previously failed here because MAX(id) was null.
        mvc.perform(post("/restful/customer/add").session(session).with(csrf()).contentType("application/json")
                .content("{\"customerName\":\"测试监护\",\"customerAge\":\"40\",\"customerSex\":\"女\",\"customerPhone\":\"13100000000\",\"customerAddress\":\"测试地址\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
        assertEquals(1,jdbc.queryForObject("select count(*) from customer",Integer.class));
        mvc.perform(post("/restful/text/add").session(session).with(csrf()).contentType("application/json")
                .content("{\"title\":\"内置数据库公告验收\",\"textMain\":\"本机记录\",\"fileListName\":[\"accepted.jpg\"]}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
        assertEquals(1,jdbc.queryForObject("select count(*) from pic",Integer.class));
        jdbc.update("insert into text(text_id,title,text_main) values(999,'无附图的历史公告','验收')");
        mvc.perform(post("/restful/text/selectDetail").session(session).with(csrf()).contentType("application/json").content("{\"textId\":999}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.pics").isEmpty());
        mvc.perform(post("/restful/text/update").session(session).with(csrf()).contentType("application/json")
                .content("{\"textId\":999,\"title\":\"更新历史公告\",\"textMain\":\"验收\",\"fileListName\":[\"new.png\"],\"picId\":-123}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
        int group=jdbc.queryForObject("select pic_id from text where text_id=999",Integer.class);
        assertTrue(group>0);
        assertEquals("new.png",jdbc.queryForObject("select pic_name from pic where pic_id=?",String.class,group));
        mvc.perform(post("/restful/text/delete").session(session).with(csrf()).contentType("application/json")
                .content("{\"textId\":999,\"picId\":-123}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
        assertEquals(0,jdbc.queryForObject("select count(*) from pic where pic_id=?",Integer.class,group));
    }
}

package com.example.careold.controller;

import com.example.careold.dao.DeviceJdbcDao;
import com.example.careold.dao.OldPersonDao;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.context.annotation.Import;
import com.example.careold.security.SecurityConfig;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;

@WebMvcTest(DeviceController.class)
@Import({com.example.careold.service.DeviceService.class, SecurityConfig.class, org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration.class})
class AnonymousAccessTests {
    @Autowired MockMvc mvc;
    @MockBean DeviceJdbcDao deviceDao;
    @MockBean com.example.careold.dao.WatchInfoDao watchInfoDao;
    @MockBean OldPersonDao oldPersonDao;
    @MockBean JdbcTemplate jdbcTemplate;

    @Test
    void anonymousVisitorCannotReadDeviceRoster() throws Exception {
        mvc.perform(post("/restful/device/list").with(csrf()).contentType("application/json").content("{}"))
                .andExpect(status().isUnauthorized());
    }
}

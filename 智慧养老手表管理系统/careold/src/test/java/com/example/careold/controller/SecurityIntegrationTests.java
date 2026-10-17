package com.example.careold.controller;

import com.example.careold.common.PasswordUtil;
import com.example.careold.common.DeviceSecretCodec;
import com.example.careold.dao.*;
import com.example.careold.domain.*;
import com.example.careold.security.*;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.context.annotation.Import;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.mock.web.MockHttpSession;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.authority.AuthorityUtils;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.test.web.servlet.MockMvc;

import java.awt.image.BufferedImage;
import java.io.ByteArrayOutputStream;
import java.nio.file.Path;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@WebMvcTest({LoginController.class, FamilyController.class, RatepreController.class, StepController.class,
        PositionController.class, DeviceController.class, ImagesApi.class, InviteCodeController.class,
        SaidController.class, UserController.class})
@Import({com.example.careold.service.DeviceService.class, SecurityConfig.class, org.springframework.boot.autoconfigure.context.PropertyPlaceholderAutoConfiguration.class})
class SecurityIntegrationTests {
    @Autowired MockMvc mvc;
    @Autowired ObjectMapper json;
    @Autowired ImagesApi images;
    @MockBean FamilyDao familyDao;
    @MockBean UserDao userDao;
    @MockBean OldPersonDao oldPersonDao;
    @MockBean RatepreDao ratepreDao;
    @MockBean StepDao stepDao;
    @MockBean PositionDao positionDao;
    @MockBean DeviceJdbcDao deviceDao;
    @MockBean WatchInfoDao watchInfoDao;
    @MockBean InviteCodeDao inviteCodeDao;
    @MockBean SaidDao saidDao;
    @MockBean NiceCountDao niceCountDao;
    @MockBean JdbcTemplate jdbcTemplate;
    @TempDir Path uploads;

    private UsernamePasswordAuthenticationToken family() {
        return UsernamePasswordAuthenticationToken.authenticated(new AccountPrincipal("FAMILY", 7, "13000000000", 1, "小明"), null,
                AuthorityUtils.createAuthorityList("ROLE_FAMILY"));
    }
    private UsernamePasswordAuthenticationToken admin() {
        return UsernamePasswordAuthenticationToken.authenticated(new AccountPrincipal("ADMIN", 2, "13100000000", null, "管理员"), null,
                AuthorityUtils.createAuthorityList("ROLE_ADMIN"));
    }
    @BeforeEach
    void setup() {
        ReflectionTestUtils.setField(images, "path", uploads.toString());
        OldPerson own = new OldPerson(); own.setOldId(1); own.setFamilyId(7);
        OldPerson other = new OldPerson(); other.setOldId(2); other.setFamilyId(8);
        when(oldPersonDao.getOldPersonById(1)).thenReturn(own);
        when(oldPersonDao.getOldPersonById(2)).thenReturn(other);
    }

    @Test
    void familyCannotInvokeAnyAdministrativeOperation() throws Exception {
        for (String endpoint : List.of("device/list", "device/create", "inviteCode/list", "inviteCode/create",
                "oldPerson/select", "family/select", "family/updateFamily", "user/select", "said/delete", "text/delete", "watchInfo/update", "ratepre/resolveAlert")) {
            mvc.perform(post("/restful/" + endpoint).with(authentication(family())).with(csrf()).contentType("application/json").content("{}"))
                    .andExpect(status().isForbidden());
        }
        verifyNoInteractions(familyDao, userDao, deviceDao, inviteCodeDao, saidDao);
    }

    @Test
    void everyFamilyRecordEndpointRejectsAnotherElder() throws Exception {
        for (String endpoint : List.of("ratepre/selectRatepreFamily", "ratepre/selectRatepreDetail", "step/selectStepFamily", "step/selectStepDetail", "position/selectFamily")) {
            mvc.perform(post("/restful/" + endpoint).with(authentication(family())).with(csrf()).contentType("application/json")
                    .content("{\"oldId\":2,\"timeFirst\":\"\",\"timeLast\":\"\"}"))
                    .andExpect(status().isForbidden());
        }
        verifyNoInteractions(ratepreDao, stepDao, positionDao);
    }

    @Test
    void ownRecordIsAccessibleUntilAdministratorUnlinksIt() throws Exception {
        mvc.perform(post("/restful/step/selectStepFamily").with(authentication(family())).with(csrf()).contentType("application/json").content("{\"oldId\":1}"))
                .andExpect(status().isOk());
        verify(stepDao).getStepSleepFamily(1);
        OldPerson unlinked = new OldPerson(); unlinked.setOldId(1); unlinked.setFamilyId(8);
        when(oldPersonDao.getOldPersonById(1)).thenReturn(unlinked);
        mvc.perform(post("/restful/step/selectStepFamily").with(authentication(family())).with(csrf()).contentType("application/json").content("{\"oldId\":1}"))
                .andExpect(status().isForbidden());
        verify(stepDao, times(1)).getStepSleepFamily(1);
    }

    @Test
    void forgedProfilePhoneAndIdAreRejectedBeforeWriting() throws Exception {
        for (String endpoint : List.of("selectTwo", "updateTwo", "updateTouPic", "updatePassword")) {
            mvc.perform(post("/restful/family/" + endpoint).with(authentication(family())).with(csrf()).contentType("application/json")
                    .content("{\"familyPhone\":\"13900000000\",\"oldPassword\":\"old-secret\",\"familyPassword\":\"new-secret\"}"))
                    .andExpect(status().isForbidden());
        }
        mvc.perform(post("/restful/family/updateTwo").with(authentication(family())).with(csrf()).contentType("application/json")
                .content("{\"familyPhone\":\"13000000000\",\"familyId\":8}"))
                .andExpect(status().isForbidden());
        verifyNoInteractions(familyDao);
    }

    @Test
    void mutationsAndLoginRequireCsrfEvenWithValidSession() throws Exception {
        mvc.perform(post("/restful/device/create").with(authentication(admin())).contentType("application/json").content("{}"))
                .andExpect(status().isForbidden()).andExpect(jsonPath("$.code").value("CSRF_REJECTED"));
        mvc.perform(post("/restful/userLogin/login").contentType("application/json").content("{}"))
                .andExpect(status().isForbidden());
        verifyNoInteractions(deviceDao, userDao);
    }

    @Test
    void loginPersistsServerIdentityRotatesSessionAndCsrfAndLogoutInvalidatesIt() throws Exception {
        Users user = new Users(); user.setUserId(2); user.setUserName("管理员"); user.setPassword(PasswordUtil.hash("good-secret"));
        when(userDao.getUsersByPhone2("13100000000")).thenReturn(List.of(user));
        var bootstrap = mvc.perform(get("/restful/userLogin/session")).andExpect(status().isOk()).andReturn();
        MockHttpSession session = (MockHttpSession) bootstrap.getRequest().getSession();
        String oldId = session.getId();
        String token = json.readTree(bootstrap.getResponse().getContentAsString()).get("csrfToken").asText();
        mvc.perform(post("/restful/userLogin/login").session(session).header("X-CSRF-TOKEN", token).contentType("application/json")
                .content("{\"phone\":\"13100000000\",\"password\":\"good-secret\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"))
                .andExpect(jsonPath("$.users1.password").doesNotExist());
        assertNotEquals(oldId, session.getId());
        var authenticated = mvc.perform(get("/restful/userLogin/session").session(session)).andExpect(jsonPath("$.account.role").value("ADMIN"))
                .andExpect(jsonPath("$.authenticated").value(true)).andReturn();
        mvc.perform(post("/restful/device/list").session(session).header("X-CSRF-TOKEN", token).contentType("application/json").content("{}"))
                .andExpect(status().isForbidden());
        String fresh = json.readTree(authenticated.getResponse().getContentAsString()).get("csrfToken").asText();
        mvc.perform(post("/restful/device/list").session(session).header("X-CSRF-TOKEN", fresh).contentType("application/json").content("{}"))
                .andExpect(status().isOk());
        mvc.perform(post("/restful/userLogin/logout").session(session).header("X-CSRF-TOKEN", fresh)).andExpect(status().isOk());
        assertTrue(session.isInvalid());
        mvc.perform(get("/restful/userLogin/session")).andExpect(jsonPath("$.authenticated").value(false));
    }

    @Test
    void invalidLoginDoesNotEstablishIdentity() throws Exception {
        mvc.perform(post("/restful/userLogin/login").with(csrf()).contentType("application/json").content("{\"phone\":\"13100000000\",\"password\":\"wrong-secret\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("0000"));
        mvc.perform(get("/restful/userLogin/session")).andExpect(jsonPath("$.authenticated").value(false));
    }

    @Test
    void familyLoginStoresOnlyItsOwnIdentityInServerSession() throws Exception {
        Family account = new Family(); account.setFamilyPassword(PasswordUtil.hash("good-secret"));
        FamilyDto family = new FamilyDto(); family.setFamilyId(7); family.setOldId(1);
        family.setFamilyPhone("13000000000"); family.setFamilyName("小明");
        when(familyDao.findByPhoneForLogin("13000000000")).thenReturn(account);
        when(familyDao.findFamilyByPhone("13000000000")).thenReturn(family);
        var login = mvc.perform(post("/restful/userLogin/login").with(csrf()).contentType("application/json")
                .content("{\"phone\":\"13000000000\",\"password\":\"good-secret\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"))
                .andExpect(jsonPath("$.family.familyPassword").doesNotExist()).andReturn();
        mvc.perform(get("/restful/userLogin/session").session((MockHttpSession) login.getRequest().getSession()))
                .andExpect(jsonPath("$.account.role").value("FAMILY")).andExpect(jsonPath("$.account.id").value(7))
                .andExpect(jsonPath("$.account.oldId").value(1)).andExpect(jsonPath("$.account.phone").value("13000000000"));
    }

    @Test
    void deviceRosterAndAccountResponsesCannotExposeSecrets() throws Exception {
        DeviceDto device = new DeviceDto(); device.setDeviceId(1); device.setSecret("real-device-secret");
        when(deviceDao.findDevices(1,20,"","all")).thenReturn(List.of(device));
        mvc.perform(post("/restful/device/list").with(authentication(admin())).with(csrf()).contentType("application/json").content("{}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.devices[0].secret").doesNotExist());
        Family family = new Family(); family.setFamilyPassword("secret");
        Users user = new Users(); user.setPassword("secret");
        assertFalse(json.writeValueAsString(family).contains("secret"));
        assertFalse(json.writeValueAsString(user).contains("secret"));
        assertFalse(json.writeValueAsString(device).contains("real-device-secret"));
        assertFalse(json.writeValueAsString(device).contains("\"secret\""));
    }

    @Test
    void hardwareStillChecksItsOwnSecretWithoutWebCsrf() throws Exception {
        mvc.perform(post("/restful/device/login").contentType("application/json").content("{\"deviceNo\":\"WTEST\",\"secret\":\"bad\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("0004"));
        verify(deviceDao).findByDeviceNo("WTEST");
    }
    @Test
    void plaintextAndMd5UsersCannotLogInButExistingShortBcryptPasswordsStillWork() throws Exception {
        Users user=new Users(); user.setUserId(2); user.setUserName("管理员");
        when(userDao.getUsersByPhone2("13100000000")).thenReturn(List.of(user));
        for (String stored : List.of("123456","e10adc3949ba59abbe56e057f20f883e")) {
            user.setPassword(stored);
            mvc.perform(post("/restful/userLogin/login").with(csrf()).contentType("application/json")
                    .content("{\"phone\":\"13100000000\",\"password\":\"123456\"}"))
                    .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("0000"));
        }
        user.setPassword(PasswordUtil.hash("123456"));
        mvc.perform(post("/restful/userLogin/login").with(csrf()).contentType("application/json")
                .content("{\"phone\":\"13100000000\",\"password\":\"123456\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
    }
    @Test
    void invalidDeviceDtosAndOversizedPagesCannotReachTheDatabase() throws Exception {
        mvc.perform(post("/restful/device/report").contentType("application/json").content("{\"secret\":\"secret\"}"))
                .andExpect(status().isBadRequest());
        mvc.perform(post("/restful/device/list").with(authentication(admin())).with(csrf()).contentType("application/json")
                .content("{\"page\":1,\"pageSize\":10000}"))
                .andExpect(status().isBadRequest());
        verifyNoInteractions(deviceDao);
    }

    @Test
    void publishedDemoSecretsCannotAuthenticateAnyHardwareOperation() throws Exception {
        Device device = new Device(); device.setDeviceId(1); device.setDeviceNo("WTEST"); device.setStatus(1); device.setSecret("demo-secret-1");
        when(deviceDao.findByDeviceNo("WTEST")).thenReturn(device);
        for (String endpoint : List.of("login","bind","heartbeat","report","sos")) {
            mvc.perform(post("/restful/device/"+endpoint).contentType("application/json").content("{\"deviceNo\":\"WTEST\",\"secret\":\"demo-secret-1\",\"code\":\"ABC123\"}"))
                    .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("0004"));
        }
        device.setSecret(DeviceSecretCodec.hash("23456789ABCDEFGHJKLMNPQRSTUVWXYZ"));
        mvc.perform(post("/restful/device/login").contentType("application/json").content("{\"deviceNo\":\"WTEST\",\"secret\":\"23456789ABCDEFGHJKLMNPQRSTUVWXYZ\"}"))
                .andExpect(status().isOk()).andExpect(jsonPath("$.returnCode").value("1111"));
    }

    @Test
    void uploadIgnoresTraversalFilenameAndRejectsNonImagesAndTraversingDownloads() throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        javax.imageio.ImageIO.write(new BufferedImage(3, 3, BufferedImage.TYPE_INT_RGB), "png", bytes);
        var uploaded = mvc.perform(multipart("/restful/image/saveImage").file(new MockMultipartFile("file", "../../escape.php", "text/plain", bytes.toByteArray()))
                .with(authentication(family())).with(csrf())).andExpect(status().isOk()).andReturn();
        String name = json.readTree(uploaded.getResponse().getContentAsString()).get("filename").asText();
        assertTrue(name.matches("[a-f0-9-]{36}\\.png"));
        assertTrue(java.nio.file.Files.isRegularFile(uploads.resolve(name)));
        mvc.perform(get("/restful/image/download").param("name", name).with(authentication(family())))
                .andExpect(status().isOk()).andExpect(content().contentType("image/png"));
        mvc.perform(multipart("/restful/image/saveImage").file(new MockMultipartFile("file", "evil.png", "image/png", "<script>alert(1)</script>".getBytes()))
                .with(authentication(family())).with(csrf())).andExpect(status().isBadRequest());
        for (String bad : List.of("../secret.png", "..\\secret.png", "C:\\secret.png", "config.xml")) {
            mvc.perform(get("/restful/image/download").param("name", bad).with(authentication(family()))).andExpect(status().isNotFound());
        }
    }

    @Test
    void oversizedBytesAndDecompressionBombDimensionsAreRejected() throws Exception {
        mvc.perform(multipart("/restful/image/saveImage").file(new MockMultipartFile("file", "big.png", "image/png", new byte[5 * 1024 * 1024 + 1]))
                .with(authentication(family())).with(csrf())).andExpect(status().isBadRequest());
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        javax.imageio.ImageIO.write(new BufferedImage(1, 1, BufferedImage.TYPE_INT_RGB), "png", bytes);
        byte[] bomb = bytes.toByteArray();
        java.nio.ByteBuffer.wrap(bomb, 16, 8).putInt(30000).putInt(30000);
        java.util.zip.CRC32 checksum = new java.util.zip.CRC32(); checksum.update(bomb, 12, 17);
        java.nio.ByteBuffer.wrap(bomb, 29, 4).putInt((int) checksum.getValue());
        mvc.perform(multipart("/restful/image/saveImage").file(new MockMultipartFile("file", "bomb.png", "image/png", bomb))
                .with(authentication(family())).with(csrf())).andExpect(status().isBadRequest());
        try (var files = java.nio.file.Files.list(uploads)) { assertEquals(0, files.count()); }
    }

    @Test
    void legacyNamedUserImageStillDownloadsFromUserDirectory() throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        javax.imageio.ImageIO.write(new BufferedImage(3, 3, BufferedImage.TYPE_INT_RGB), "jpg", bytes);
        java.nio.file.Files.write(uploads.resolve("kai.jpg"), bytes.toByteArray());
        mvc.perform(get("/restful/image/download").param("name", "kai.jpg").with(authentication(family())))
                .andExpect(status().isOk()).andExpect(content().contentType("image/jpeg"))
                .andExpect(content().bytes(bytes.toByteArray()));
    }
}

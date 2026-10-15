package com.example.careold.controller;

import com.example.careold.common.PasswordUtil;
import com.example.careold.dao.FamilyDao;
import com.example.careold.domain.Family;
import java.util.LinkedHashMap;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.AfterEach;
import com.example.careold.security.AccountPrincipal;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.core.authority.AuthorityUtils;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

class FamilyPasswordTests {
    private FamilyController controller;
    private FamilyDao dao;

    @BeforeEach
    void setUp() {
        dao = mock(FamilyDao.class);
        controller = new FamilyController();
        controller.familyDao = dao;
        SecurityContextHolder.getContext().setAuthentication(UsernamePasswordAuthenticationToken.authenticated(
                new AccountPrincipal("FAMILY", 7, "13000000000", 1, "小明"), null, AuthorityUtils.createAuthorityList("ROLE_FAMILY")));
    }

    @AfterEach
    void clearSession() { SecurityContextHolder.clearContext(); }

    private LinkedHashMap request(String oldPassword, String newPassword) {
        LinkedHashMap params = new LinkedHashMap();
        params.put("familyPhone", "13000000000");
        params.put("oldPassword", oldPassword);
        params.put("familyPassword", newPassword);
        return params;
    }

    private void account(String storedPassword) {
        Family family = new Family();
        family.setFamilyPassword(storedPassword);
        when(dao.findByPhoneForLogin("13000000000")).thenReturn(family);
    }

    @Test
    void missingOrNonStringFieldsCannotReachDatabase() {
        assertEquals("0000", controller.updatePassword(null).get("returnCode"));
        assertEquals("0000", controller.updatePassword(new LinkedHashMap()).get("returnCode"));
        LinkedHashMap params = request("old-secret", "new-secret");
        params.put("oldPassword", 123456);
        assertEquals("0000", controller.updatePassword(params).get("returnCode"));
        verifyNoInteractions(dao);
    }

    @Test
    void invalidNewPasswordAndEmptyOldPasswordAreRejected() {
        for (String value : new String[]{"", "short", "123456789012345678901"}) {
            assertEquals("0000", controller.updatePassword(request("old-secret", value)).get("returnCode"));
        }
        assertEquals("0000", controller.updatePassword(request("", "new-secret")).get("returnCode"));
        verifyNoInteractions(dao);
    }

    @Test
    void unknownPhoneCannotChangePassword() {
        assertEquals("0000", controller.updatePassword(request("old-secret", "new-secret")).get("returnCode"));
        verify(dao, never()).updateFamilyPasswordByPhoneForFamily(anyString(), anyString());
    }

    @Test
    void incorrectOldPasswordCannotChangeBcryptAccount() {
        account(PasswordUtil.hash("old-secret"));
        assertEquals("0000", controller.updatePassword(request("wrong-secret", "new-secret")).get("returnCode"));
        verify(dao, never()).updateFamilyPasswordByPhoneForFamily(anyString(), anyString());
    }

    @Test
    void correctOldPasswordCanChangeBcryptAccount() {
        account(PasswordUtil.hash("old-secret"));
        when(dao.updateFamilyPasswordByPhoneForFamily("new-secret", "13000000000")).thenReturn(1);
        assertEquals("1111", controller.updatePassword(request("old-secret", "new-secret")).get("returnCode"));
        verify(dao).updateFamilyPasswordByPhoneForFamily("new-secret", "13000000000");
    }

    @Test
    void legacyPasswordStillRequiresCorrectOldPassword() {
        account("legacy-secret");
        assertEquals("0000", controller.updatePassword(request("wrong-secret", "new-secret")).get("returnCode"));
        when(dao.updateFamilyPasswordByPhoneForFamily("new-secret", "13000000000")).thenReturn(1);
        assertEquals("1111", controller.updatePassword(request("legacy-secret", "new-secret")).get("returnCode"));
        verify(dao, times(1)).updateFamilyPasswordByPhoneForFamily("new-secret", "13000000000");
    }

    @Test
    void databaseNotUpdatedIsNotReportedAsSuccess() {
        account("old-secret");
        assertEquals("0000", controller.updatePassword(request("old-secret", "new-secret")).get("returnCode"));
    }
}

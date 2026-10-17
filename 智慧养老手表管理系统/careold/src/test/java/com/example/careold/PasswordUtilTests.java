package com.example.careold;

import com.example.careold.common.PasswordUtil;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

public class PasswordUtilTests {

    @Test
    public void bcryptHashMatchesOnlyItsPassword() {
        String hash = PasswordUtil.hash("care-password-2026");

        assertTrue(PasswordUtil.isHashed(hash));
        assertTrue(PasswordUtil.matches("care-password-2026", hash));
        assertFalse(PasswordUtil.matches("incorrect-password", hash));
    }

    @Test
    public void plaintextAndLegacyDigestsCannotAuthenticate() {
        assertFalse(PasswordUtil.matches("legacy-password", "legacy-password"));
        assertFalse(PasswordUtil.matches("123456", "e10adc3949ba59abbe56e057f20f883e"));
        assertFalse(PasswordUtil.matches("e10adc3949ba59abbe56e057f20f883e", "e10adc3949ba59abbe56e057f20f883e"));
        assertFalse(PasswordUtil.isHashed("legacy-password"));
        assertFalse(PasswordUtil.matches("incorrect-password", "legacy-password"));
    }

    @Test
    public void bcryptVersionsCannotBeUsedAsPlaintextPasswords() {
        String original=PasswordUtil.hash("care-password-2026");
        for (String version : new String[]{"$2a$", "$2b$", "$2y$"}) {
            String hash=version+original.substring(4);
            assertFalse(PasswordUtil.matches(hash,hash));
            assertTrue(PasswordUtil.isHashed(hash));
            assertTrue(PasswordUtil.matches("care-password-2026",hash));
        }
        assertFalse(PasswordUtil.matches("$2b$broken", "$2b$broken"));
    }
}

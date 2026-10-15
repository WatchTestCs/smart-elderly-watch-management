package com.example.careold.security;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class PublicRateLimitTests {
    @Test
    void exhaustedClientCannotEvictOtherClientsOrBypassItsWindow() {
        var limiter = new PublicRateLimitFilter();
        assertTrue(limiter.allow("first", 2, 0));
        assertTrue(limiter.allow("first", 2, 1));
        assertFalse(limiter.allow("first", 2, 2));
        for (int i=1; i<10000; i++) assertTrue(limiter.allow("client-"+i, 2, 3));
        assertFalse(limiter.allow("overflow", 2, 4));
        assertFalse(limiter.allow("first", 2, 5));
        assertTrue(limiter.allow("overflow", 2, 600010));
        assertTrue(limiter.allow("first", 2, 600011));
    }
}

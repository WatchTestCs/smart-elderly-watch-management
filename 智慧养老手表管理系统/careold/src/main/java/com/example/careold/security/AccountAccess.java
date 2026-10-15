package com.example.careold.security;

import com.example.careold.dao.OldPersonDao;
import com.example.careold.domain.OldPerson;
import org.springframework.http.HttpStatus;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.server.ResponseStatusException;

public final class AccountAccess {
    private AccountAccess() { }

    public static AccountPrincipal current() {
        Authentication authentication = SecurityContextHolder.getContext().getAuthentication();
        if (authentication == null || !(authentication.getPrincipal() instanceof AccountPrincipal account)) {
            throw new ResponseStatusException(HttpStatus.UNAUTHORIZED, "请先登录");
        }
        return account;
    }

    public static String phone(Object requested) {
        AccountPrincipal account = current();
        if ("ADMIN".equals(account.role())) return String.valueOf(requested);
        if (requested != null && !account.phone().equals(requested.toString())) forbidden();
        return account.phone();
    }

    public static void familyId(Object requested) {
        AccountPrincipal account = current();
        if (!"ADMIN".equals(account.role()) && !String.valueOf(account.id()).equals(String.valueOf(requested))) forbidden();
    }

    public static String oldId(Object requested, OldPersonDao dao) {
        AccountPrincipal account = current();
        String value = requested == null ? String.valueOf(account.oldId()) : requested.toString();
        int id;
        try { id = Integer.parseInt(value); }
        catch (NumberFormatException e) { throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "老人编号不正确"); }
        if (!"ADMIN".equals(account.role())) {
            // 每次核对当前数据库归属，管理员解除关联后旧会话立即失去读取权限。
            OldPerson old = dao.getOldPersonById(id);
            if (old == null || !account.id().equals(old.getFamilyId())) forbidden();
        }
        return value;
    }

    private static void forbidden() {
        throw new ResponseStatusException(HttpStatus.FORBIDDEN, "无权访问该账户或照护记录");
    }
}

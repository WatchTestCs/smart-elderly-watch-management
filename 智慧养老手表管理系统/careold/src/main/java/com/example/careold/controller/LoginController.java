package com.example.careold.controller;

import com.example.careold.common.PasswordUtil;
import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.dao.FamilyDao;
import com.example.careold.dao.UserDao;
import com.example.careold.domain.Family;
import com.example.careold.domain.FamilyDto;
import com.example.careold.domain.Users;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.List;
import java.util.Map;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import com.example.careold.security.AccountPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.authority.AuthorityUtils;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.web.context.HttpSessionSecurityContextRepository;
import org.springframework.security.web.csrf.CsrfToken;
import org.springframework.security.web.csrf.CsrfTokenRepository;
import org.springframework.security.web.csrf.CsrfAuthenticationStrategy;
import org.springframework.security.web.authentication.session.ChangeSessionIdAuthenticationStrategy;

@Controller
@RequestMapping("/restful/userLogin")
public class LoginController {

    @Autowired
    private FamilyDao familyDao;

    @Autowired
    private UserDao userDao;

    @Autowired
    private HttpSessionSecurityContextRepository contexts;
    @Autowired
    private CsrfTokenRepository tokens;

    @ResponseBody
    @GetMapping("/session")
    public Map<String, Object> session(HttpServletRequest request) {
        CsrfToken csrf = (CsrfToken) request.getAttribute(CsrfToken.class.getName());
        var authentication = SecurityContextHolder.getContext().getAuthentication();
        boolean authenticated = authentication != null && authentication.getPrincipal() instanceof AccountPrincipal;
        return Map.of("authenticated", authenticated, "setupRequired", !userDao.hasAdministrators(), "account", authenticated ? authentication.getPrincipal() : Map.of(),
                "csrfToken", csrf.getToken(), "csrfHeader", csrf.getHeaderName());
    }

    private void establish(AccountPrincipal account, HttpServletRequest request, HttpServletResponse response) {
        var authentication = UsernamePasswordAuthenticationToken.authenticated(account, null,
                AuthorityUtils.createAuthorityList("ROLE_" + account.role()));
        new ChangeSessionIdAuthenticationStrategy().onAuthentication(authentication, request, response);
        new CsrfAuthenticationStrategy(tokens).onAuthentication(authentication, request, response);
        var context = SecurityContextHolder.createEmptyContext();
        context.setAuthentication(authentication);
        SecurityContextHolder.setContext(context);
        contexts.saveContext(context, request, response);
    }

    @ResponseBody
    @PostMapping("/login")
    public LinkedHashMap login(@RequestBody LinkedHashMap param, HttpServletRequest request, HttpServletResponse response) {
        LinkedHashMap result = new LinkedHashMap();
        Object phoneValue = param == null ? null : param.get("phone");
        Object passwordValue = param == null ? null : param.get("password");
        if (phoneValue == null || passwordValue == null) {
            return failure(result, "请输入手机号和密码");
        }

        String phone = phoneValue.toString().trim();
        String password = passwordValue.toString();
        if (!phone.matches("1\\d{10}") || password.isEmpty() || password.length() > 72 || password.getBytes(java.nio.charset.StandardCharsets.UTF_8).length > 72) {
            return failure(result, "请输入手机号和密码");
        }

        List<Users> users = userDao.getUsersByPhone2(phone);
        if (!users.isEmpty() && PasswordUtil.matches(password, users.get(0).getPassword())) {
            Users user = users.get(0);
            user.setPassword(null);
            establish(new AccountPrincipal("ADMIN", user.getUserId(), phone, null, user.getUserName()), request, response);
            result.put("users1", user);
            result.put(ReturnCodeUtil.returnCode, ReturnCodeUtil.successCode);
            return result;
        }

        Family familyAccount = familyDao.findByPhoneForLogin(phone);
        if (familyAccount != null && PasswordUtil.matches(password, familyAccount.getFamilyPassword())) {
            FamilyDto family = familyDao.findFamilyByPhone(phone);
            if (family == null) {
                return failure(result, "该家属账号尚未关联老人，请联系管理员");
            }
            establish(new AccountPrincipal("FAMILY", family.getFamilyId(), phone, family.getOldId(), family.getFamilyName()), request, response);
            result.put("family", family);
            result.put(ReturnCodeUtil.returnCode, ReturnCodeUtil.successCode);
            return result;
        }

        return failure(result, "手机号或密码不正确");
    }

    private LinkedHashMap failure(LinkedHashMap result, String message) {
        result.put(ReturnCodeUtil.returnCode, ReturnCodeUtil.falseCode);
        result.put("message", message);
        return result;
    }
}

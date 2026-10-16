package com.example.careold.controller;

import com.example.careold.dao.UserDao;
import com.example.careold.domain.Users;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.List;

@Controller
@RequestMapping("/restful/user")
public class UserController {

    @Autowired
    UserDao userDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap selectText(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String userPhone=param.get("userPhone").toString();
        List<Users> usersByPhone2 = userDao.getUsersByPhone2(userPhone);
        usersByPhone2.forEach(user -> user.setPassword(null));
        result.put("usersByPhone2",usersByPhone2);
        return result;
    }
}

package com.example.careold.controller;

import com.example.careold.dao.OldPersonDetailDao;
import com.example.careold.domain.OldPersonDetail;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;


@Controller
@RequestMapping("/restful/oldPerson")
public class OldPersonDetailController {

    @Autowired
    OldPersonDetailDao oldPersonDetailDao;

    @ResponseBody
    @PostMapping("/selectDetail")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String name=param.get("name").toString();
        oldPersonDetailDao.getOldPersonDetail(name,com.example.careold.domain.PageRequest.from(param)).putInto(result,"oldPersonDetails");
        return result;

    }
}

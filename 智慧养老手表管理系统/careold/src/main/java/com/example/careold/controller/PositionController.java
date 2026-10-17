package com.example.careold.controller;


import com.example.careold.dao.PositionDao;
import com.example.careold.domain.PositionDto;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import com.example.careold.security.AccountAccess;
import com.example.careold.dao.OldPersonDao;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.List;

@Controller
@RequestMapping("/restful/position")
public class PositionController {

    @Autowired
    OldPersonDao oldPersonDao;

    @Autowired
    PositionDao positionDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String name=param.get("name").toString();
        positionDao.getPosition(name,com.example.careold.domain.PageRequest.from(param)).putInto(result,"positionDtos");
        return result;
    }

    @ResponseBody
    @PostMapping("/selectFamily")
    public LinkedHashMap selectFamily(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String oldId=AccountAccess.oldId(param.get("oldId"), oldPersonDao);
        List<PositionDto> positionDtos=positionDao.getPositionFamily(Integer.parseInt(oldId));
        result.put("positionDtos",positionDtos);
        return result;

    }

}

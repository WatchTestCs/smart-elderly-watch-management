package com.example.careold.controller;

import com.example.careold.dao.AlarmProcessDao;
import com.example.careold.domain.AlarmProcess;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@Controller
@RequestMapping("/restful/alarmProcess")
public class AlarmProcessController {

    @Autowired
    AlarmProcessDao alarmProcessDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap params){
        LinkedHashMap result=new LinkedHashMap();
        String id=(String)params.get("id");
        List<AlarmProcess> alarmProcesses= alarmProcessDao.getAlarmProcess(Integer.parseInt(id));
        result.put("alarmProcesses",alarmProcesses);
        return result;
    }
}

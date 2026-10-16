package com.example.careold.controller;

import com.example.careold.dao.RatepreDao;
import com.example.careold.dao.StepDao;
import com.example.careold.domain.StepDto;
import java.util.LinkedHashMap;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import com.example.careold.security.AccountAccess;
import com.example.careold.dao.OldPersonDao;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.ArrayList;
import java.util.List;

@Controller
@RequestMapping("/restful/step")
public class StepController {

    @Autowired
    OldPersonDao oldPersonDao;

    @Autowired
    StepDao stepDao;

    @Autowired
    RatepreDao ratepreDao;

    private static final Logger LOG = LoggerFactory.getLogger(RatepreController.class);

    //睡眠步数查询
    @ResponseBody
    @PostMapping("/selectStep")
    public LinkedHashMap selectRatepre(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String name=param.get("name").toString();
        String timeFirst=param.get("timeFirst").toString();
        String timeLast=param.get("timeLast").toString();
        LOG.info(name+"----"+timeFirst+"----"+timeLast);
        stepDao.getStepSleep(name,timeFirst,timeLast,com.example.careold.domain.PageRequest.from(param)).putInto(result,"stepDtos");
        return result;
    }

    @ResponseBody
    @PostMapping("/selectStepFamily")
    public LinkedHashMap selectStepFamily(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String oldId=AccountAccess.oldId(param.get("oldId"), oldPersonDao);
        List<StepDto> stepDtos=stepDao.getStepSleepFamily(Integer.parseInt(oldId));
        result.put("stepDtos",stepDtos);
        return result;
    }


    @ResponseBody
    @PostMapping("/selectStepDetail")
    public LinkedHashMap selectStepDetail(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String oldId=AccountAccess.oldId(param.get("oldId"), oldPersonDao);
        List<StepDto> stepDtos=stepDao.getStepSleepDetail(Integer.parseInt(oldId));
        ArrayList sleep=new ArrayList();
        ArrayList createTime=new ArrayList();
        for(int i=0;i<stepDtos.size();i++){
            sleep.add(stepDtos.get(i).getSleep());
            createTime.add(stepDtos.get(i).getCreateTime().substring(8,10));
        }
        result.put("stepDtos",stepDtos);
        result.put("sleep",sleep);
        result.put("createTime",createTime);
        return result;
    }

    @ResponseBody
    @PostMapping("/selectStepCount")
    public LinkedHashMap selectStepCount(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        List<String> deviceIdList=stepDao.getSleepBadCount();
        result.put("deviceIdList",deviceIdList);
        return result;
    }

    @ResponseBody
    @PostMapping("/getAllCount")
    public LinkedHashMap getAllCount(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        List<String> rates=ratepreDao.getRateCount();
        List<String> pres=ratepreDao.getPreCount();
        List<String> deviceIdList=stepDao.getSleepBadCount();
        String allCount=rates.size()+pres.size()+deviceIdList.size()+"";
        result.put("allCount",allCount);
        return result;
    }

}

package com.example.careold.controller;


import com.example.careold.common.DateConst;
import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.dao.OldPersonDao;
import com.example.careold.dao.DeviceDao;
import com.example.careold.dao.InviteCodeDao;
import com.example.careold.dao.WatchInfoDao;
import com.example.careold.domain.OldPerson;
import com.example.careold.domain.WatchDto;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.http.HttpStatus;
import org.springframework.web.server.ResponseStatusException;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.Date;
import java.util.List;

@Controller
@RequestMapping("/restful/oldPerson")
public class OldPersonController {

    @Autowired
    OldPersonDao oldPersonDao;

    @Autowired
    WatchInfoDao watchInfoDao;

    @Autowired
    InviteCodeDao inviteCodeDao;

    @Autowired
    DeviceDao deviceDao;

    //对老人的查询
    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String name=param.get("name").toString();
        oldPersonDao.getOldPersonPage(name,com.example.careold.domain.PageRequest.from(param)).putInto(result,"oldPeople");
        return result;
    }

    @ResponseBody
    @PostMapping("/add")
    public LinkedHashMap add(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String deviceId=param.get("deviceId").toString();
        List<WatchDto> watchInfo2 = watchInfoDao.getWatchInfo2(Integer.parseInt(deviceId));
        if(watchInfo2.size()==1){
            String name=param.get("name").toString();
            String bothdate=param.get("bothdate").toString();
            String bothnian1=bothdate.substring(0,4);
            String sex=param.get("sex").toString();
            String phone=param.get("phone").toString();
            String room=param.get("room").toString();
            String beTime= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_5);
            String bothnian2=beTime.substring(0,4);
            String illness=param.get("illness").toString();
            String warn=param.get("warn").toString();
            String foodLike=param.get("foodLike").toString();
            String foodHate=param.get("foodHate").toString();
            String drag=param.get("drag").toString();
            String customerId=param.get("customerId").toString();
            OldPerson oldPerson=new OldPerson();
            oldPerson.setName(name);
            oldPerson.setAge(Integer.parseInt(bothnian2)-Integer.parseInt(bothnian1));
            oldPerson.setBeTime(beTime);
            oldPerson.setSex(sex);
            oldPerson.setPhone(phone);
            oldPerson.setRoom(room);
            oldPerson.setIllness(illness);
            oldPerson.setWarn(warn);
            oldPerson.setFoodLike(foodLike);
            oldPerson.setFoodHate(foodHate);
            oldPerson.setDrag(drag);
            oldPerson.setBothdate(bothdate);
            oldPerson.setDeviceId(Integer.parseInt(deviceId));
            oldPerson.setCustomerId(Integer.parseInt(customerId));
            int rows=oldPersonDao.addOldPerson(oldPerson);
            if(rows==1){
                result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
                return result;
            }
        }

        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/update")
    public LinkedHashMap update(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String name=param.get("name").toString();
        String bothdate=param.get("bothdate").toString();
        String bothnian1=bothdate.substring(0,4);
        String sex=param.get("sex").toString();
        String phone=param.get("phone").toString();
        String room=param.get("room").toString();
        String beTime= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_1);
        String bothnian2=beTime.substring(0,4);
        String illness=param.get("illness").toString();
        String warn=param.get("warn").toString();
        String foodLike=param.get("foodLike").toString();
        String foodHate=param.get("foodHate").toString();
        String drag=param.get("drag").toString();
        String oldId=param.get("oldId").toString();
        String deviceId=param.get("deviceId").toString();
        String customerId=param.get("customerName").toString();
        if(!ReturnCodeUtil.isNumeric(customerId)){
            customerId=param.get("customerId").toString();
        }
        OldPerson oldPerson=new OldPerson();
        oldPerson.setName(name);
        oldPerson.setAge(Integer.parseInt(bothnian2)-Integer.parseInt(bothnian1));
//        oldPerson.setBeTime(beTime);
        oldPerson.setSex(sex);
        oldPerson.setPhone(phone);
        oldPerson.setRoom(room);
        oldPerson.setIllness(illness);
        oldPerson.setWarn(warn);
        oldPerson.setFoodLike(foodLike);
        oldPerson.setFoodHate(foodHate);
        oldPerson.setDrag(drag);
        oldPerson.setBothdate(bothdate);
        oldPerson.setOldId(Integer.parseInt(oldId));
        oldPerson.setDeviceId(Integer.parseInt(deviceId));
        oldPerson.setCustomerId(Integer.parseInt(customerId));
        int rows=oldPersonDao.updateOldPerson(oldPerson);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/delete")
    @Transactional
    public LinkedHashMap delete(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String oldId=param.get("oldId").toString();
        // 老人删除后，其名下未使用的邀请码一并作废，避免出现无主邀请码
        if (oldPersonDao.getOldPersonById(Integer.parseInt(oldId)) == null) {
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
            return result;
        }
        // A live watch association must be explicitly unbound before removing
        // the elder. Do not rely on an opaque SQL foreign-key error or cascade.
        if (deviceDao.countByOldId(Integer.parseInt(oldId)) > 0) {
            throw new ResponseStatusException(HttpStatus.CONFLICT, "请先解除该长者名下设备绑定");
        }
        inviteCodeDao.deleteByOldId(Integer.parseInt(oldId));
        int rows=oldPersonDao.deleteOldPerson(Integer.parseInt(oldId));
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        throw new IllegalStateException("长者删除失败，已撤销邀请码清理");
    }

}

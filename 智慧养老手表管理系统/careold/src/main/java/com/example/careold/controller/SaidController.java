package com.example.careold.controller;

import com.example.careold.common.DateConst;
import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.dao.NiceCountDao;
import com.example.careold.dao.SaidDao;
import com.example.careold.domain.Said;
import com.example.careold.domain.Said2;
import com.example.careold.domain.SaidDto;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import com.example.careold.security.AccountAccess;
import com.example.careold.dao.FamilyDao;
import org.springframework.transaction.annotation.Transactional;
import java.util.Date;
import java.util.List;

@Controller
@RequestMapping("/restful/said")
public class SaidController {

    @Autowired
    SaidDao saidDao;

    @Autowired
    NiceCountDao niceCountDao;

    @Autowired
    FamilyDao familyDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String textId=param.get("textId").toString();
        String familyPhone=AccountAccess.current().phone();
        List<Said2> said2s=saidDao.getSaidWithLikes(Integer.parseInt(textId),familyPhone);
        result.put("said2s",said2s);
        return result;

    }

    @ResponseBody
    @PostMapping("/selectByUser")
    public LinkedHashMap selectByUser(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        List<SaidDto> saidDtos=saidDao.getSaidForAdmin();
        result.put("saidDtos",saidDtos);
        return result;

    }


    @ResponseBody
    @PostMapping("/delete")
    public LinkedHashMap delete(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String saidId=param.get("saidId").toString();
        int rows=saidDao.deleteSaid(Integer.parseInt(saidId));
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;

    }

    @ResponseBody
    @PostMapping("/insert")
    public LinkedHashMap insert(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String said=param.get("said").toString();
        String textId=param.get("textId").toString();
        String personName=AccountAccess.current().displayName();
        String phone=AccountAccess.current().phone();
        var profile=familyDao.getFamilyByPhoneForFamily(phone);
        String touPic=profile == null ? "" : profile.getFamilyTouPic();
        String time= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_1);
        Said said1=new Said();
        said1.setSaid(said);
        said1.setTextId(Integer.parseInt(textId));
        said1.setPersonName(personName);
        said1.setSaidPhone(phone);
        said1.setSaidTime(time);
        said1.setTouPic(touPic);
        int rows=saidDao.addSaid(said1);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;

    }


    @ResponseBody
    @PostMapping("/update")
    public LinkedHashMap update(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String back=param.get("back").toString();
        String saidId=param.get("saidId").toString();
        Said said1=new Said();
        said1.setSaidId(Integer.parseInt(saidId));
        said1.setBack(back);
        String backTime= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_1);
        said1.setBackTime(backTime);
        int rows=saidDao.updateSaid(said1);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;

    }



    @ResponseBody
    @PostMapping("/goodCount")
    @Transactional
    public LinkedHashMap goodCount(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        int rows=0;
        String saidId=param.get("saidId").toString();
        String phone=AccountAccess.current().phone();
        if (!saidDao.lockById(Integer.parseInt(saidId))) throw new org.springframework.web.server.ResponseStatusException(org.springframework.http.HttpStatus.NOT_FOUND, "评论不存在");
        List<String> saidList=niceCountDao.getCount(Integer.parseInt(saidId),phone);
        if(saidList.size()>0){
            rows=niceCountDao.deleteCount(Integer.parseInt(saidId),phone);
            saidDao.updateSaid(saidId,false);
        }else{
            rows=niceCountDao.addCount(Integer.parseInt(saidId),phone);
            saidDao.updateSaid(saidId,true);
        }

        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;

    }


}

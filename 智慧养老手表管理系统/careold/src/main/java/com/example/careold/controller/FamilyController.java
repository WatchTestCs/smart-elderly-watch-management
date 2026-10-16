package com.example.careold.controller;

import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.security.AccountAccess;
import com.example.careold.common.PasswordUtil;
import com.example.careold.dao.FamilyDao;
import com.example.careold.dao.OldPersonDao;
import com.example.careold.domain.*;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.List;

@Controller
@RequestMapping("/restful/family")
public class FamilyController {

    @Autowired
    FamilyDao familyDao;

    @Autowired
    OldPersonDao oldPersonDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String familyName=param.get("familyName").toString();
        familyDao.findFamily(familyName,com.example.careold.domain.PageRequest.from(param)).putInto(result,"familyDtos");
        return result;
    }


    @ResponseBody
    @PostMapping("/add")
    @Transactional
    public LinkedHashMap add(@RequestBody LinkedHashMap params){
        LinkedHashMap result=new LinkedHashMap();
        String familyName=params.get("familyName").toString();
        String familyRelation=params.get("familyRelation").toString();
        String familySex=params.get("familySex").toString();
        String familyPhone=params.get("familyPhone").toString();
        String oldPhone=params.get("oldPhone").toString();
        String familyAddress=params.get("familyAddress").toString();


        Family family=new Family();
        family.setFamilyAddress(familyAddress);
        family.setFamilyPhone(familyPhone);
        family.setFamilySex(familySex);
        family.setFamilyName(familyName);
        family.setFamilyRelation(familyRelation);
        int row=familyDao.save(family);
        Integer oldId=familyDao.findOldIdByPhone(oldPhone);
        String familyId=familyDao.findFamilyIdByPhone(familyPhone);
        if (row != 1 || oldPersonDao.updateOldPerson2(oldId,Integer.parseInt(familyId)) != 1) {
            throw new IllegalStateException("家属关联失败，已撤销本次新增");
        }
        if(row==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/updateFamily")
    public LinkedHashMap updateFamily(@RequestBody LinkedHashMap params){
        LinkedHashMap result=new LinkedHashMap();
        String familyName=params.get("familyName").toString();
        String familyRelation=params.get("familyRelation").toString();
        String familySex=params.get("familySex").toString();
        String familyPhone=params.get("familyPhone").toString();
        String familyAddress=params.get("familyAddress").toString();
        String familyId=params.get("familyId").toString();
        Family family=new Family();
        family.setFamilyAddress(familyAddress);
        family.setFamilyPhone(familyPhone);
        family.setFamilySex(familySex);
        family.setFamilyName(familyName);
        family.setFamilyRelation(familyRelation);
        family.setFamilyId(Integer.parseInt(familyId));
        int row=familyDao.update(family);
        if(row==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/deleteFamily")
    @Transactional
    public LinkedHashMap deleteFamily(@RequestBody LinkedHashMap params){
        LinkedHashMap result=new LinkedHashMap();
        String familyId=params.get("familyId").toString();
        int row=familyDao.delete(familyId);
        if(row==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/selectTwo")
    public LinkedHashMap selectTwo(@RequestBody LinkedHashMap params){
        AccountAccess.phone(params.get("familyPhone"));
        LinkedHashMap result=new LinkedHashMap();
        String familyPhone=params.get("familyPhone").toString();
        FamilyDto2 familyDto2 =familyDao.getFamilyByPhoneForFamily(familyPhone);

        result.put("familyDto2",familyDto2);
        return result;
    }

    @ResponseBody
    @PostMapping("/updateTwo")
    public LinkedHashMap updateTwo(@RequestBody LinkedHashMap params){
        AccountAccess.phone(params.get("familyPhone"));
        AccountAccess.familyId(params.get("familyId"));
        LinkedHashMap result=new LinkedHashMap();
        String familyPhone=params.get("familyPhone").toString();
        String familyName=params.get("familyName").toString();
        String familySex=params.get("familySex").toString();
        String familyId=params.get("familyId").toString();
        String familyAddress=params.get("familyAddress").toString();
        FamilyDto2 familyDto2=new FamilyDto2();
        familyDto2.setFamilyAddress(familyAddress);
        familyDto2.setFamilyName(familyName);
        familyDto2.setFamilySex(familySex);
        familyDto2.setFamilyPhone(familyPhone);
        familyDto2.setFamilyId(Integer.parseInt(familyId));
        int rows=familyDao.updateFamilyByPhoneForFamily(familyDto2);

        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/updateTouPic")
    public LinkedHashMap updateTouPic(@RequestBody LinkedHashMap params){
        AccountAccess.phone(params.get("familyPhone"));
        LinkedHashMap result=new LinkedHashMap();
        String familyPhone=params.get("familyPhone").toString();
        String familyTouPic=params.get("familyTouPic").toString();
        int rows=familyDao.updateFamilyTouPic(familyTouPic,familyPhone);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/updatePassword")
    public LinkedHashMap updatePassword(@RequestBody LinkedHashMap params){
        LinkedHashMap result=new LinkedHashMap();
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        if (params == null || !(params.get("familyPhone") instanceof String)
                || !(params.get("oldPassword") instanceof String)
                || !(params.get("familyPassword") instanceof String)) {
            result.put("message", "请完整填写手机号、原密码和新密码");
            return result;
        }
        String familyPhone=AccountAccess.phone(params.get("familyPhone"));
        String oldPassword=(String) params.get("oldPassword");
        String familyPassword=(String) params.get("familyPassword");
        if (familyPhone.trim().isEmpty() || oldPassword.isEmpty()
                || !PasswordUtil.validNewPassword(familyPassword)) {
            result.put("message", "请填写原密码，新密码须为12–20位且包含字母和数字");
            return result;
        }
        Family account=familyDao.findByPhoneForLogin(familyPhone);
        if (account == null || !PasswordUtil.matches(oldPassword, account.getFamilyPassword())) {
            result.put("message", "手机号或原密码不正确");
            return result;
        }
        int rows=familyDao.updateFamilyPasswordByPhoneForFamily(familyPassword,familyPhone);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

}

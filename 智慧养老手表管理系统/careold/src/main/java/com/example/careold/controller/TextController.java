package com.example.careold.controller;

import com.example.careold.common.DateConst;
import com.example.careold.common.DefineData;
import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.common.SmsUtil;
import com.example.careold.dao.FamilyDao;
import com.example.careold.dao.PicDao;
import com.example.careold.dao.TextDao;
import com.example.careold.domain.FamilyDto;
import com.example.careold.domain.Pic;
import com.example.careold.domain.Text;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.Date;
import java.util.List;

@Controller
@RequestMapping("/restful/text")
public class TextController {

    @Autowired
    private SmsUtil smsUtil;

    @Autowired
    TextDao textDao;

    @Autowired
    PicDao picDao;

    @Autowired
    FamilyDao familyDao;

    //公告查询
    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap selectText(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String title=param.get("title").toString();
        List<Text> texts = textDao.selectTextAll(title);
        result.put("texts",texts);
        return result;
    }

    @ResponseBody
    @PostMapping("/selectDetail")
    public LinkedHashMap selectTextDetail(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String textId=param.get("textId").toString();
        Text text=textDao.selectTextDetail(Integer.parseInt(textId));
        List<Pic> pics=picDao.getPic(text.getPicId());
        result.put("text",text);
        result.put("pics",pics);
        return result;
    }

    //添加公告文章接口
    @ResponseBody
    @PostMapping("/add")
    @org.springframework.transaction.annotation.Transactional
    public LinkedHashMap addText(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String title=param.get("title").toString();
        String textMain=param.get("textMain").toString();
        List fileList = (List)param.get("fileListName");
        String time= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_1);
        int picId=textDao.nextPicId();
        Pic pic1=new Pic();
        for(int i=0;i<fileList.size();i++){
            pic1.setPicId(picId);
            pic1.setPicName(fileList.get(i).toString());
            picDao.addPic(pic1);
        }
        Text text=new Text();
        text.setTitle(title);
        text.setTextMain(textMain);
        text.setTime(time);
        text.setPicId(picId);
        int rows=textDao.addText(text);
        if(rows==1){
            List<FamilyDto> familyDtos=familyDao.findFamily(null);
            for(int i=0;i<familyDtos.size();i++){
                smsUtil.sendMessage(familyDtos.get(i).getFamilyPhone());
            }
            List<Text> texts = textDao.selectTextAll(null);
            result.put("texts",texts);
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/delete")
    @org.springframework.transaction.annotation.Transactional
    public LinkedHashMap deleteText(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String textId=param.get("textId").toString();
        String picId=param.get("picId").toString();
        int rows=textDao.deleteText(Integer.parseInt(textId));
        picDao.deletePic(Integer.parseInt(picId));
        if(rows==1){
            List<Text> texts = textDao.selectTextAll(null);
            result.put("texts",texts);
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/update")
    @org.springframework.transaction.annotation.Transactional
    public LinkedHashMap updateText(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String title=param.get("title").toString();
        String textMain=param.get("textMain").toString();
        String time= DateConst.dateToString(new Date(),DateConst.DATE_MODEL_1);
        String textId=param.get("textId").toString();
        String picId=param.get("picId").toString();
        List fileList=(List)param.get("fileListName");
        picDao.deletePic(Integer.parseInt(picId));
        Pic pic1=new Pic();
        for(int i=0;i<fileList.size();i++){
            pic1.setPicId(Integer.parseInt(picId));
            pic1.setPicName(fileList.get(i).toString());
            picDao.addPic(pic1);
        }
        Text text=new Text();
        text.setTitle(title);
        text.setTextMain(textMain);
        text.setTime(time);
        text.setTextId(Integer.parseInt(textId));
        int rows=textDao.update(text);
        if(rows==1){
            List<Text> texts = textDao.selectTextAll(null);
            result.put("texts",texts);
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }
}

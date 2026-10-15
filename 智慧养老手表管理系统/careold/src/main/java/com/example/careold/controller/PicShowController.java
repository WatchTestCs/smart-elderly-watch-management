package com.example.careold.controller;

import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.dao.NiceCountDao;
import com.example.careold.dao.PicShowDao;
import com.example.careold.domain.PicShow;
import com.example.careold.domain.Said2;
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
@RequestMapping("/restful/picshow")
public class PicShowController {

    @Autowired
    PicShowDao picShowDao;

    @ResponseBody
    @PostMapping("/add")
    @Transactional
    public LinkedHashMap add(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        if (param == null || !(param.get("fileListName") instanceof List)) return result;
        List fileList=(List) param.get("fileListName");
        if (fileList.isEmpty() || fileList.size() > 10) return result;
        for (Object value : fileList) {
            if (!(value instanceof String)) return result;
            String filename=(String) value;
            if (filename.trim().isEmpty() || filename.contains("/") || filename.contains("\\")) return result;
        }
        picShowDao.delete(null);
        for (Object value : fileList) {
            PicShow picShow=new PicShow();
            picShow.setPicName((String) value);
            if (picShowDao.save(picShow) != 1) throw new IllegalStateException("展示图片保存失败");
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        List<PicShow> picShows= picShowDao.findById(null);
        System.out.println(picShows);
        result.put("picShows",picShows);
        return result;

    }

}

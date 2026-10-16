package com.example.careold.controller;

import com.example.careold.common.DateConst;
import com.example.careold.common.ReturnCodeUtil;
import com.example.careold.dao.WatchInfoDao;
import com.example.careold.domain.WatchDto;
import com.example.careold.domain.WatchInfo;
import java.util.LinkedHashMap;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseBody;

import org.springframework.http.HttpStatus;
import org.springframework.web.server.ResponseStatusException;
import java.util.Date;
import java.util.List;

@Controller
@RequestMapping("/restful/watchInfo")
public class WatchInfoController {

    @Autowired
    WatchInfoDao watchInfoDao;

    @ResponseBody
    @PostMapping("/select")
    public LinkedHashMap select(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String oldName=param.get("oldName").toString();
        watchInfoDao.getWatchInfo(oldName,com.example.careold.domain.PageRequest.from(param)).putInto(result,"watchDtos");
        return result;
    }

    @ResponseBody
    @PostMapping("/add")
    public LinkedHashMap add(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        WatchInfo watchInfo = profile(param);
        List<WatchDto> watchInfo2 = watchInfoDao.getWatchInfo2(watchInfo.getDeviceId());
        if(watchInfo2.size()==0){
            watchInfo.setOpTime(DateConst.dateToString(new Date(),DateConst.DATE_MODEL_5));
            int rows=watchInfoDao.addWatchInfo(watchInfo);
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
        WatchInfo watchInfo = profile(param);
        int rows=watchInfoDao.updateWatchInfo(watchInfo);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    @ResponseBody
    @PostMapping("/delete")
    public LinkedHashMap delete(@RequestBody LinkedHashMap param){
        LinkedHashMap result=new LinkedHashMap();
        String deviceId=param.get("deviceId").toString();

        int rows=watchInfoDao.deleteWatchInfo(deviceId);
        if(rows==1){
            result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.successCode);
            return result;
        }
        result.put(ReturnCodeUtil.returnCode,ReturnCodeUtil.falseCode);
        return result;
    }

    private WatchInfo profile(LinkedHashMap param) {
        WatchInfo info = new WatchInfo();
        info.setDeviceId(requiredNumber(param, "deviceId", 1, Integer.MAX_VALUE));
        info.setHrf(requiredNumber(param, "hrf", 20, 250));
        info.setHrl(requiredNumber(param, "hrl", 20, 250));
        info.setBpf(requiredNumber(param, "bpf", 40, 300));
        info.setBpl(requiredNumber(param, "bpl", 40, 300));
        if (info.getHrl() >= info.getHrf() || info.getBpl() >= info.getBpf())
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "低值必须小于高值");
        return info;
    }

    private int requiredNumber(LinkedHashMap param, String key, int min, int max) {
        try {
            Object raw = param == null ? null : param.get(key);
            int value = Integer.parseInt(raw == null ? "" : raw.toString().trim());
            if (value >= min && value <= max) return value;
        } catch (NumberFormatException ignored) { }
        throw new ResponseStatusException(HttpStatus.BAD_REQUEST, key + " 缺失或超出允许范围");
    }

}

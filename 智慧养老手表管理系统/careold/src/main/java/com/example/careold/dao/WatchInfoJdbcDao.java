package com.example.careold.dao;

import com.example.careold.domain.WatchDto;
import com.example.careold.domain.WatchInfo;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class WatchInfoJdbcDao implements WatchInfoDao {

    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public PageResult<WatchDto> getWatchInfo(String oldName, PageRequest page) {
        String value=oldName==null?"":oldName;
        return PagedQuery.query(jdbcTemplate,"SELECT w.device_id,w.op_time,w.hrf,w.bpf,w.hrl,w.bpl,o.name FROM watch_info w LEFT JOIN oldperson o ON w.device_id=o.device_id WHERE (?='' OR o.name LIKE ?) ORDER BY w.device_id",WatchDto.class,page,value,"%"+value+"%");
    }

    @Override
    public List<WatchDto> getWatchInfo2(int deviceId) {
        String sql;
        List<WatchDto> watchDtos;
        sql="select device_id,op_time,hrf,bpf,hrl,bpl from watch_info where device_id=?";
        watchDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(WatchDto.class),deviceId);
        return watchDtos;
    }

    @Override
    public int addWatchInfo(WatchInfo watchInfo) {
        String sql="insert into watch_info(device_id,op_time,bpf,bpl,hrl,hrf) values(?,?,?,?,?,?)";
        return jdbcTemplate.update(sql,watchInfo.getDeviceId(),watchInfo.getOpTime(),watchInfo.getBpf(),watchInfo.getBpl(),watchInfo.getHrl(),watchInfo.getHrf());
    }

    @Override
    public int updateWatchInfo(WatchInfo watchInfo) {
        String sql="update watch_info set bpf=?,bpl=?,hrl=?,hrf=? where device_id=?";
        return jdbcTemplate.update(sql,watchInfo.getBpf(),watchInfo.getBpl(),watchInfo.getHrl(),watchInfo.getHrf(),watchInfo.getDeviceId());
    }

    @Override
    public int deleteWatchInfo(String deviceId) {
        String sql="delete from watch_info where device_id=?";
        return jdbcTemplate.update(sql,deviceId);
    }
}

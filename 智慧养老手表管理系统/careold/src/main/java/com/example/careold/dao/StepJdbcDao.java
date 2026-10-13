package com.example.careold.dao;

import com.example.careold.domain.RatepreDto;
import com.example.careold.domain.StepDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class StepJdbcDao implements StepDao {

    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public List<StepDto>  getStepSleep(String name, String timeFirst, String timeLast) {
        String sql;
        List<StepDto> stepDtos=null;
        if((name==null || "".equals(name)) && (timeFirst==null || "".equals(timeFirst)) && (timeFirst==null || "".equals(timeFirst))){
            sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,step,sleep,sleep_sate,create_time FROM oldperson left JOIN watch_info ON oldperson.device_id=watch_info.device_id left JOIN watch_day_info ON watch_info.`device_id`=watch_day_info.device_id where sleep>0  GROUP BY oldperson.old_id  ORDER BY watch_day_info.create_time DESC";
            stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class));
        }else if(name==null || "".equals(name)){
            sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,step,sleep,sleep_sate,create_time FROM oldperson left JOIN watch_info ON oldperson.device_id=watch_info.device_id left JOIN watch_day_info ON watch_info.`device_id`=watch_day_info.device_id WHERE create_time BETWEEN ? AND ?  GROUP BY oldperson.old_id  ORDER BY watch_day_info.create_time DESC";
            stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),timeFirst,timeLast);
        }else if((timeFirst==null || "".equals(timeFirst)) && (timeFirst==null || "".equals(timeFirst))){
            sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,step,sleep,sleep_sate,create_time FROM oldperson left JOIN watch_info ON oldperson.device_id=watch_info.device_id left JOIN watch_day_info ON watch_info.`device_id`=watch_day_info.device_id WHERE oldperson.name like ?  GROUP BY oldperson.old_id  ORDER BY watch_day_info.create_time DESC";
            stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),"%"+name+"%");
        }

        return stepDtos;
    }

    @Override
    public List<StepDto> getStepSleepFamily(int oldId) {
        String sql;
        List<StepDto> stepDtos=null;
        sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,step,sleep,sleep_sate,create_time FROM oldperson left JOIN watch_info ON oldperson.device_id=watch_info.device_id left JOIN watch_day_info ON watch_info.`device_id`=watch_day_info.device_id WHERE oldperson.old_id=?  GROUP BY oldperson.old_id  ORDER BY watch_day_info.create_time DESC";
        stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),oldId);

        return stepDtos;
    }

    @Override
    public List<String> getSleepBadCount() {
        String sql="select device_id deviceId from watch_day_info where sleep_sate='1' group by device_id";
        List<String> deviceIdList=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(String.class));
        return deviceIdList;
    }

    @Override
    public List<StepDto> getStepSleepDetail(int oldId) {
        String sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,step,sleep,sleep_sate,create_time FROM oldperson left JOIN watch_info ON oldperson.device_id=watch_info.device_id left JOIN watch_day_info ON watch_info.`device_id`=watch_day_info.device_id WHERE oldperson.old_id=? ORDER BY watch_day_info.create_time ASC LIMIT 5";
        List<StepDto> stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),oldId);
        return stepDtos;
    }
}

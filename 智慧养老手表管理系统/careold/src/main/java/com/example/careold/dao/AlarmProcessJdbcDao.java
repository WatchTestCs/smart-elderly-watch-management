package com.example.careold.dao;

import com.example.careold.domain.AlarmProcess;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class AlarmProcessJdbcDao implements AlarmProcessDao{

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Override
    public List<AlarmProcess> getAlarmProcess(int id) {
        String sql;
        List<AlarmProcess> alarmProcesses;
        if(id==0){
            sql="SELECT * FROM alarm_process";

            alarmProcesses=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(AlarmProcess.class));

        }else{
            sql="SELECT * FROM alarm_process WHERE id=?";
            alarmProcesses=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(AlarmProcess.class),id);
        }

        return alarmProcesses;
    }
}

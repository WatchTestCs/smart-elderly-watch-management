package com.example.careold.dao;

import com.example.careold.domain.PositionDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class PositionJdbcDao implements PositionDao {

    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public List<PositionDto> getPosition(String name) {
        String sql=null;
        List<PositionDto> positionDtos=null;
        if(name==null || "".equals(name)){
            sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,lng,lat,alarm_precess.be_time,alarm_precess.status FROM oldperson left JOIN alarm_precess ON oldperson.`old_id`=alarm_precess.`old_id` GROUP BY oldperson.old_id  ORDER BY alarm_precess.be_time DESC limit 3";
            positionDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(PositionDto.class));
        }else{
            sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,lng,lat,alarm_precess.be_time,alarm_precess.status FROM oldperson left JOIN alarm_precess ON oldperson.`old_id`=alarm_precess.`old_id` WHERE name=? GROUP BY oldperson.old_id  ORDER BY alarm_precess.be_time DESC";
            positionDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(PositionDto.class),name);
        }
        return positionDtos;
    }

    @Override
    public List<PositionDto> getPositionFamily(int oldId) {
        String sql=null;
        List<PositionDto> positionDtos=null;
        sql="SELECT oldperson.`name`,oldperson.old_id,oldperson.`sex`,lng,lat,alarm_precess.be_time,alarm_precess.status FROM oldperson left JOIN alarm_precess ON oldperson.`old_id`=alarm_precess.`old_id` WHERE oldperson.`old_id`=? GROUP BY oldperson.old_id  ORDER BY alarm_precess.be_time DESC";
        positionDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(PositionDto.class),oldId);
        return positionDtos;
    }

}

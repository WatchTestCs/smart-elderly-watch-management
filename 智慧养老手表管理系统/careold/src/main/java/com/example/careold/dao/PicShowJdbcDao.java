package com.example.careold.dao;

import com.example.careold.domain.PicShow;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class PicShowJdbcDao extends JdbcCom implements PicShowDao{


    @Override
    public List<PicShow> findById(String s) {
        String sql="select * from pic_show";
        return jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(PicShow.class));
    }

    @Override
    public int save(PicShow picShow) {
        String sql="insert into pic_show(pic_name) values(?)";
        return jdbcTemplate.update(sql,picShow.getPicName());
    }

    @Override
    public int delete(String s) {
        String sql="delete from pic_show";
        return jdbcTemplate.update(sql);
    }
}

package com.example.careold.dao;

import com.example.careold.domain.Pic;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class PicJdbcDao implements PicDao{

    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public List<Pic> getPic(int picId) {
        String sql="SELECT * FROM pic WHERE pic_id=?";
        return jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(Pic.class),picId);
    }

    @Override
    public int addPic(Pic pic) {
        String sql="INSERT INTO pic(pic_id,pic_name) VALUES(?,?)";
        return jdbcTemplate.update(sql,pic.getPicId(),pic.getPicName());
    }

    @Override
    public int deletePic(int picId) {
        String sql="DELETE FROM pic WHERE pic_id=?";
        return jdbcTemplate.update(sql,picId);
    }
}

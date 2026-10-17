package com.example.careold.dao;

import com.example.careold.domain.Said;
import com.example.careold.domain.Said2;
import com.example.careold.domain.SaidDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class SaidJdbcDao implements SaidDao {

    @Override
    public List<Said2> getSaidWithLikes(int textId, String phone) {
        String sql = "select s.*, case when exists (select 1 from nice_count n where n.said_id=s.said_id and n.phone=?) then '1' else '0' end as good from said s";
        return textId == 0 ? jdbcTemplate.query(sql, new BeanPropertyRowMapper<>(Said2.class), phone)
                : jdbcTemplate.query(sql + " where s.text_id=?", new BeanPropertyRowMapper<>(Said2.class), phone, textId);
    }

    @Override
    public boolean lockById(int saidId) {
        return !jdbcTemplate.queryForList("select said_id from said where said_id=? for update", Integer.class, saidId).isEmpty();
    }


    @Autowired
    JdbcTemplate jdbcTemplate;



    @Override
    public com.example.careold.domain.PageResult<SaidDto> getSaidForAdmin(com.example.careold.domain.PageRequest page) {
        String sql="select said_id,said,back,said_time,back_time,person_name,tou_pic,title from said inner join text on said.text_id=text.text_id order by said_time desc,said_id desc";
        return PagedQuery.query(jdbcTemplate,sql,SaidDto.class,page);
    }

    @Override
    public int addSaid(Said said) {
        String sql="INSERT INTO said(said,text_id,said_time,person_name,said_phone,tou_pic) VALUES(?,?,?,?,?,?)";
        return jdbcTemplate.update(sql,said.getSaid(),said.getTextId(),said.getSaidTime(),said.getPersonName(),said.getSaidPhone(),said.getTouPic());
    }

    @Override
    public int deleteSaid(int saidId) {
        String sql="DELETE FROM said WHERE said_id=?";
        return jdbcTemplate.update(sql,saidId);
    }

    @Override
    public int updateSaid(Said said) {
        String sql="UPDATE said SET back=?,back_time=? WHERE said_id=?";
        return jdbcTemplate.update(sql,said.getBack(),said.getBackTime(),said.getSaidId());
    }

    @Override
    public int updateSaid(String saidId,boolean flag) {
        String sql;
        if(flag){
            sql="update said set nice_count=nice_count+1 where said_id=?";
        }else{
            sql="update said set nice_count=nice_count-1 where said_id=?";
        }
        return jdbcTemplate.update(sql,saidId);
    }


}

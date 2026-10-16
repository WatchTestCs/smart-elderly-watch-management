package com.example.careold.dao;

import com.example.careold.domain.Text;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class TextJdbcDao implements TextDao {

    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public PageResult<Text> selectTextAll(String title, PageRequest page) {
        String value=title==null?"":title;
        return PagedQuery.query(jdbcTemplate,"SELECT * FROM text WHERE (?='' OR title LIKE ?) ORDER BY time DESC,text_id DESC",Text.class,page,value,"%"+value+"%");
    }


    @Override
    public Text selectTextDetail(int textId) {
        String sql="SELECT * FROM text WHERE text_id=?";
        Text text=jdbcTemplate.queryForObject(sql,new BeanPropertyRowMapper<>(Text.class),textId);
        return text;
    }

    @Override
    public int addText(Text text) {
        String sql="INSERT INTO text(title,text_main,TIME,pic_id) VALUES(?,?,?,?)";
        return jdbcTemplate.update(sql,text.getTitle(),text.getTextMain(),text.getTime(),text.getPicId());
    }

    @Override
    public int deleteText(int id) {
        String sql="DELETE FROM text WHERE text_id=?";
        return jdbcTemplate.update(sql,id);
    }

    @Override
    public int update(Text text) {
        String sql="UPDATE text SET title=?,text_main=?,TIME=?,pic_id=? WHERE text_id=?";
        return jdbcTemplate.update(sql,text.getTitle(),text.getTextMain(),text.getTime(),text.getPicId(),text.getTextId());
    }

    @Override
    public int nextPicId() {
        String sql="SELECT NEXT VALUE FOR picture_group";
        return jdbcTemplate.queryForObject(sql,int.class);
    }
}

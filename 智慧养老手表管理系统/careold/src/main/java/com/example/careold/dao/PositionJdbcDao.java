package com.example.careold.dao;

import com.example.careold.domain.PositionDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;
import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class PositionJdbcDao implements PositionDao {
    @Autowired JdbcTemplate jdbcTemplate;

    // Rank each source once, then join one row per elder.
    private static final String LATEST = "SELECT o.name,o.old_id AS oldId,o.sex,l.lng,l.lat,"
            + " GREATEST(COALESCE(l.be_time,o.be_time),COALESCE(a.be_time,o.be_time)) AS beTime,"
            + " COALESCE(a.status,'0') AS status FROM oldperson o"
            + " LEFT JOIN (SELECT old_id,lng,lat,be_time,ROW_NUMBER() OVER(PARTITION BY old_id ORDER BY id DESC) AS rn FROM device_location) l"
            + " ON l.old_id=o.old_id AND l.rn=1"
            + " LEFT JOIN (SELECT old_id,status,be_time,ROW_NUMBER() OVER(PARTITION BY old_id ORDER BY be_time DESC,id DESC) AS rn FROM alarm_precess) a"
            + " ON a.old_id=o.old_id AND a.rn=1";

    @Override public PageResult<PositionDto> getPosition(String name, PageRequest page) {
        String sql=LATEST+" WHERE (?='' OR o.name LIKE ?) ORDER BY beTime DESC,o.old_id";
        String value=name==null?"":name;
        return PagedQuery.query(jdbcTemplate,sql,PositionDto.class,page,value,"%"+value+"%");
    }

    @Override public List<PositionDto> getPositionFamily(int oldId) {
        String sql = "SELECT o.name,o.old_id AS oldId,o.sex,l.lng,l.lat,"
                + " GREATEST(COALESCE(l.be_time,o.be_time),COALESCE(a.be_time,o.be_time)) AS beTime,"
                + " COALESCE(a.status,'0') AS status FROM oldperson o"
                + " LEFT JOIN (SELECT old_id,lng,lat,be_time FROM device_location WHERE old_id=? ORDER BY id DESC LIMIT 1) l"
                + " ON l.old_id=o.old_id"
                + " LEFT JOIN (SELECT old_id,status,be_time FROM alarm_precess WHERE old_id=? ORDER BY be_time DESC,id DESC LIMIT 1) a"
                + " ON a.old_id=o.old_id WHERE o.old_id=?";
        return jdbcTemplate.query(sql, new BeanPropertyRowMapper<>(PositionDto.class), oldId, oldId, oldId);
    }
}

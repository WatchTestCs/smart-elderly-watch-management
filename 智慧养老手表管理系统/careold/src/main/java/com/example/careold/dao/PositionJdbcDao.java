package com.example.careold.dao;

import com.example.careold.domain.PositionDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;
import java.util.List;

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

    @Override public List<PositionDto> getPosition(String name) {
        var mapper = new BeanPropertyRowMapper<>(PositionDto.class);
        if (name == null || name.isEmpty()) return jdbcTemplate.query(LATEST + " ORDER BY beTime DESC", mapper);
        return jdbcTemplate.query(LATEST + " WHERE o.name LIKE ? ORDER BY beTime DESC", mapper, "%" + name + "%");
    }

    @Override public List<PositionDto> getPositionFamily(int oldId) {
        return jdbcTemplate.query(LATEST + " WHERE o.old_id=?", new BeanPropertyRowMapper<>(PositionDto.class), oldId);
    }
}

package com.example.careold.dao;

import com.example.careold.domain.StepDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class StepJdbcDao implements StepDao {

    @Autowired
    JdbcTemplate jdbcTemplate;

    /** 每台设备按日期和主键确定最新日记录。 */
    private static final String SELECT_FIELDS =
            "SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, w.step AS step, w.sleep AS sleep,"
            + " w.sleep_sate AS sleepSate, FORMATDATETIME(w.create_time,'yyyy-MM-dd') AS createTime"
            + " FROM oldperson o";

    private static final String LATEST_SELECT = SELECT_FIELDS
            + " INNER JOIN (SELECT *,ROW_NUMBER() OVER(PARTITION BY device_id ORDER BY create_time DESC,id DESC) AS rn FROM watch_day_info) w"
            + " ON w.device_id=o.device_id AND w.rn=1";

    @Override
    public PageResult<StepDto> getStepSleep(String name, String timeFirst, String timeLast, PageRequest page) {
        String sql="SELECT * FROM (" + LATEST_SELECT + ") t WHERE 1=1";
        var values = new java.util.ArrayList<Object>();
        if (name != null && !name.isBlank()) { sql += " AND t.name LIKE ?"; values.add("%"+name+"%"); }
        if (timeFirst != null && !timeFirst.isBlank() && timeLast != null && !timeLast.isBlank()) {
            sql += " AND t.createTime BETWEEN ? AND ?"; values.add(timeFirst); values.add(timeLast);
        }
        return PagedQuery.query(jdbcTemplate,sql+" ORDER BY t.createTime DESC,t.oldId",StepDto.class,page,values.toArray());
    }

    @Override
    public List<StepDto> getStepSleepFamily(int oldId) {
        String sql;
        List<StepDto> stepDtos;
        sql=SELECT_FIELDS
                + " INNER JOIN watch_day_info w ON w.device_id=o.device_id WHERE o.old_id=?"
                + " ORDER BY w.create_time DESC,w.id DESC LIMIT 1";
        stepDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),oldId);

        return stepDtos;
    }

    @Override
    public List<String> getSleepBadCount() {
        String sql="select device_id deviceId from watch_day_info where sleep_sate='1' group by device_id";
        return jdbcTemplate.queryForList(sql,String.class);
    }

    @Override
    public List<StepDto> getStepSleepDetail(int oldId) {
        String sql="SELECT r.name, r.oldId, r.sex, r.step, r.sleep, r.sleepSate,"
                + " FORMATDATETIME(r.create_time,'yyyy-MM-dd') AS createTime FROM ("
                + " SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, w.step AS step, w.sleep AS sleep,"
                + " w.sleep_sate AS sleepSate, w.create_time, w.id"
                + " FROM oldperson o INNER JOIN watch_day_info w ON w.device_id = o.device_id"
                + " WHERE o.old_id=? ORDER BY w.create_time DESC, w.id DESC LIMIT 7) r"
                + " ORDER BY r.create_time ASC, r.id ASC";
        return jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(StepDto.class),oldId);
    }
}

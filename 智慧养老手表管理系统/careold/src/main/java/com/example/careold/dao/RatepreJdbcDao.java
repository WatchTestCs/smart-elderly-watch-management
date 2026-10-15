package com.example.careold.dao;

import com.example.careold.domain.RatepreDto;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Map;
@Repository
public class RatepreJdbcDao implements RatepreDao{

    @Autowired
    JdbcTemplate jdbcTemplate;

    /**
     * 说明：原来的 SQL 依赖 MySQL 5.x 的宽松 GROUP BY，取不到真正最新的一条记录，
     * 而且列表里 hr/bp 一直是 0。这里改成「先取每个老人最新一条记录的时间，再回表取该条数据」。
     */
    private static final String LATEST_SELECT =
            "SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, r.rate_state AS rateState,"
            + " r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,"
            + " r.rate_handled AS rateHandled, r.pre_handled AS preHandled,"
            + " FORMATDATETIME(r.create_time,'yyyy-MM-dd HH:mm:ss') AS createTime"
            + " FROM oldperson o"
            + " INNER JOIN (SELECT *,ROW_NUMBER() OVER(PARTITION BY old_id ORDER BY create_time DESC,id DESC) AS rn FROM ratepre) r"
            + " ON r.old_id=o.old_id AND r.rn=1";

    @Override
    public List<RatepreDto> getRatepre(String name, String timeFirst, String timeLast) {
        String sql;
        List<RatepreDto> ratepreDtos=null;
        boolean noName = (name==null || "".equals(name));
        boolean noTime = (timeFirst==null || "".equals(timeFirst)) || (timeLast==null || "".equals(timeLast));
        if(noName && noTime){
            sql="SELECT * FROM (" + LATEST_SELECT + ") t ORDER BY t.createTime DESC";
            ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class));
        }else if(noName){
            sql="SELECT * FROM (" + LATEST_SELECT + ") t WHERE t.createTime BETWEEN ? AND ? ORDER BY t.createTime DESC";
            ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class),timeFirst,timeLast);
        }else{
            sql="SELECT * FROM (" + LATEST_SELECT + ") t WHERE t.name like ? ORDER BY t.createTime DESC";
            ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class),"%"+name+"%");
        }
        return ratepreDtos;
    }

    @Override
    public List<RatepreDto> getRatepreFamily(int oldId, String timeFirst, String timeLast) {
        String sql;
        List<RatepreDto> ratepreDtos=null;
        if(timeFirst==null || "".equals(timeFirst) || timeLast==null || "".equals(timeLast)){
            sql="SELECT * FROM (" + LATEST_SELECT + ") t WHERE t.oldId=? ORDER BY t.createTime DESC";
            ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class),oldId);
        }else{
            sql="SELECT * FROM (" + LATEST_SELECT + ") t WHERE t.createTime BETWEEN ? AND ? AND t.oldId=?"
                    + " ORDER BY t.createTime DESC";
            ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class),timeFirst,timeLast,oldId);
        }
        return ratepreDtos;
    }

    @Override
    public List<RatepreDto> getRatepreDetial(int oldId) {
        String sql="SELECT r.hr AS hr, r.bp AS bp, FORMATDATETIME(r.create_time,'yyyy-MM-dd HH:mm:ss') AS createTime"
                + " FROM (SELECT id, hr, bp, create_time FROM ratepre WHERE old_id=?"
                + " ORDER BY create_time DESC, id DESC LIMIT 30) r ORDER BY r.create_time ASC, r.id ASC";
        List<RatepreDto> ratepreDtos=null;
        ratepreDtos=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(RatepreDto.class),oldId);
        return ratepreDtos;
    }

    @Override
    public List<String> getRateCount() {
        String sql="SELECT device_id deviceId FROM ratepre WHERE rate_state='1' GROUP BY device_id";
        List<String> deviceids=null;
        deviceids=jdbcTemplate.queryForList(sql,String.class);
        return deviceids;
    }

    @Override
    public List<String> getPreCount() {
        String sql="SELECT device_id deviceId FROM ratepre WHERE pre_state='1' GROUP BY device_id";
        List<String> deviceids=null;
        deviceids=jdbcTemplate.queryForList(sql,String.class);
        return deviceids;
    }

    // ==================== 异常（红点）相关 ====================

    @Override
    public int countPendingAlert(String kind) {
        boolean isRate = !"pre".equals(kind);
        String stateColumn = isRate ? "rate_state" : "pre_state";
        String handledAtColumn = isRate ? "rate_handled_at" : "pre_handled_at";
        // 只统计"上次处理之后"新产生的异常，处理过的历史异常不再重复亮红点
        String sql = "select count(1) from ratepre where " + stateColumn + " = '1'"
                + " and (" + handledAtColumn + " is null or create_time > " + handledAtColumn + ")";
        Integer count = jdbcTemplate.queryForObject(sql, Integer.class);
        return count == null ? 0 : count;
    }

    @Override
    public Map<String, Object> findAlertRecord(int recordId) {
        String sql = "select r.id as id, r.device_id as deviceId, r.old_id as oldId,"
                + " r.hr as hr, r.bp as bp, r.rate_state as rateState, r.pre_state as preState,"
                + " r.rate_handled as rateHandled, r.pre_handled as preHandled,"
                + " (r.rate_handled_at is not null and r.create_time <= r.rate_handled_at) as rateDone,"
                + " (r.pre_handled_at is not null and r.create_time <= r.pre_handled_at) as preDone,"
                + " FORMATDATETIME(r.create_time,'yyyy-MM-dd HH:mm:ss') as createTime, o.name as oldPersonName"
                + " from ratepre r left join oldperson o on o.old_id = r.old_id where r.id = ?";
        List<Map<String, Object>> rows = jdbcTemplate.queryForList(sql, recordId);
        return rows.isEmpty() ? null : rows.get(0);
    }

    @Override
    public Map<String, Object> findLatestPendingByOldId(int oldId, String kind) {
        String condition;
        if ("rate".equals(kind)) {
            condition = "(r.rate_state = '1' and (r.rate_handled_at is null or r.create_time > r.rate_handled_at))";
        } else if ("pre".equals(kind)) {
            condition = "(r.pre_state = '1' and (r.pre_handled_at is null or r.create_time > r.pre_handled_at))";
        } else {
            condition = "((r.rate_state = '1' and (r.rate_handled_at is null or r.create_time > r.rate_handled_at))"
                    + " or (r.pre_state = '1' and (r.pre_handled_at is null or r.create_time > r.pre_handled_at)))";
        }
        String sql = "select r.id as id, r.device_id as deviceId, r.old_id as oldId,"
                + " r.hr as hr, r.bp as bp, r.rate_state as rateState, r.pre_state as preState,"
                + " r.rate_handled as rateHandled, r.pre_handled as preHandled,"
                + " FORMATDATETIME(r.create_time,'yyyy-MM-dd HH:mm:ss') as createTime, o.name as oldPersonName"
                + " from ratepre r left join oldperson o on o.old_id = r.old_id"
                + " where r.old_id = ? and " + condition
                + " order by r.create_time desc, r.id desc limit 1";
        List<Map<String, Object>> rows = jdbcTemplate.queryForList(sql, oldId);
        return rows.isEmpty() ? null : rows.get(0);
    }

    @Override
    public int resolveAlert(int recordId, String kind) {
        boolean isRate = !"pre".equals(kind);
        String stateColumn = isRate ? "rate_state" : "pre_state";
        String handledColumn = isRate ? "rate_handled" : "pre_handled";
        String handledAtColumn = isRate ? "rate_handled_at" : "pre_handled_at";
        return jdbcTemplate.update("update ratepre set " + handledColumn + " = 1, " + handledAtColumn
                + " = NOW(), " + stateColumn + " = '0' where id = ?", recordId);
    }


}

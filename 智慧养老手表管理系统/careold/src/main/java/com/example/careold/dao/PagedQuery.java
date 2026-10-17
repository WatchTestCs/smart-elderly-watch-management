package com.example.careold.dao;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;
import java.util.Arrays;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.core.JdbcTemplate;

final class PagedQuery {
    private PagedQuery() { }
    static <T> PageResult<T> query(JdbcTemplate jdbc, String sql, Class<T> type, PageRequest page, Object... args) {
        long total=jdbc.queryForObject("select count(*) from ("+sql+") paged_total",Long.class,args);
        Object[] bounded=Arrays.copyOf(args,args.length+2);
        bounded[args.length]=page.pageSize(); bounded[args.length+1]=page.offset();
        var items=jdbc.query(sql+" LIMIT ? OFFSET ?",new BeanPropertyRowMapper<>(type),bounded);
        return new PageResult<>(items,total,page.page(),page.pageSize());
    }
}

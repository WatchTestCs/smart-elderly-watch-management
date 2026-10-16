package com.example.careold.dao;

import com.example.careold.domain.OldPersonDetail;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class OldPersonDetailJdbcDao implements OldPersonDetailDao{


    @Autowired
    JdbcTemplate jdbcTemplate;

    @Override
    public PageResult<OldPersonDetail> getOldPersonDetail(String name, PageRequest page) {
        String value=name==null?"":name;
        return PagedQuery.query(jdbcTemplate,"SELECT o.*,f.family_name,f.family_relation,f.family_phone,f.family_sex,f.family_address,c.customer_name FROM oldperson o LEFT JOIN family f ON f.family_id=o.family_id LEFT JOIN customer c ON c.customer_id=o.customer_id WHERE (?='' OR o.name LIKE ?) ORDER BY o.old_id",OldPersonDetail.class,page,value,"%"+value+"%");
    }

}

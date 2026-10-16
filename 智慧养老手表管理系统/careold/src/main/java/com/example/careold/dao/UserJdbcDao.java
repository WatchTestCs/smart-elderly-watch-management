package com.example.careold.dao;

import com.example.careold.domain.Users;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public class UserJdbcDao extends JdbcCom implements UserDao {
    @Override
    public boolean hasAdministrators() {
        return jdbcTemplate.queryForObject("SELECT COUNT(*) FROM users", Integer.class) > 0;
    }






    @Override
    public List<Users> getUsersByPhone2(String phone) {

        String sql="Select user_id,user_name,password,role,user_phone from users where user_phone = ?";
        return jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(Users.class),phone);
    }

}

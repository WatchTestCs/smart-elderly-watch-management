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
    public List<Users> findById(String s) {
        String sql="";
        List<Users> users=null;
        if(s==null || "".equals(s)){
            sql="select user_id,user_name,role,user_phone from users";
            users=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(Users.class));
        }else{
            sql="select user_id,user_name,role,user_phone from users where user_name like ?";
            users=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(Users.class),"%"+s+"%");
        }
        return users;
    }





    @Override
    public List<Users> getUsersByPhone2(String phone) {

        String sql="Select user_id,user_name,password,role,user_phone from users where user_phone = ?";
        List<Users> users=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(Users.class),phone);
        return users;
    }

    @Override
    public int updatePasswordByPhone(String password, String phone) {
        String sql = "update users set password=? where user_phone=?";
        return jdbcTemplate.update(sql, password, phone);
    }
}

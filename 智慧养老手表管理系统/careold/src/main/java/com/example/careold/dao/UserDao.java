package com.example.careold.dao;

import com.example.careold.domain.Users;

import java.util.List;

public interface UserDao {
    boolean hasAdministrators();

    List<Users> findById(String name);
    List<Users> getUsersByPhone2(String phone);

    int updatePasswordByPhone(String password, String phone);
}

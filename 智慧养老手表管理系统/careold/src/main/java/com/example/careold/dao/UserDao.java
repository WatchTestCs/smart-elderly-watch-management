package com.example.careold.dao;

import com.example.careold.domain.Users;

import java.util.List;

public interface UserDao {
    boolean hasAdministrators();

    List<Users> getUsersByPhone2(String phone);

}

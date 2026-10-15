package com.example.careold.dao;


import com.example.careold.domain.PicShow;

public interface PicShowDao {
    java.util.List<PicShow> findById(String id);
    int save(PicShow image);
    int delete(String id);

}

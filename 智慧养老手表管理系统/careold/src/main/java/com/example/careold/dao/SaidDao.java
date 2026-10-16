package com.example.careold.dao;

import com.example.careold.domain.Said;
import com.example.careold.domain.Said2;
import com.example.careold.domain.SaidDto;

import java.util.List;

public interface SaidDao {
    List<Said2> getSaidWithLikes(int textId, String phone);

    boolean lockById(int saidId);

    com.example.careold.domain.PageResult<SaidDto> getSaidForAdmin(com.example.careold.domain.PageRequest page);

    int addSaid(Said said);

    int deleteSaid(int saidId);

    int updateSaid(Said said);


    int updateSaid(String saidId,boolean flag);
}

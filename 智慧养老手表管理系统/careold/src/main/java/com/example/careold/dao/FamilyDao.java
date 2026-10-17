package com.example.careold.dao;

import com.example.careold.domain.Family;
import com.example.careold.domain.FamilyDto;
import com.example.careold.domain.FamilyDto2;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface FamilyDao{
    List<String> notificationPhones();

    int save(Family family);
    int update(Family family);
    int delete(String id);

    public PageResult<FamilyDto> findFamily(String familyName, PageRequest page);

    FamilyDto findFamilyByPhone(String phone);

    Family findByPhoneForLogin(String phone);

    Integer findOldIdByPhone(String phone);

    String findFamilyIdByPhone(String phone);

    FamilyDto2 getFamilyByPhoneForFamily(String phone);

    int updateFamilyTouPic(String pic,String phone);

    int updateFamilyByPhoneForFamily(FamilyDto2 familyDto2);

    int updateFamilyPasswordByPhoneForFamily(String password,String phone);


    /**
     * 新增家属并返回自增主键 family_id（失败返回 null）
     */
    Integer saveAndGetId(Family family);

    /**
     * 统计该手机号是否已存在家属账号
     */
    int countByPhone(String phone);

}

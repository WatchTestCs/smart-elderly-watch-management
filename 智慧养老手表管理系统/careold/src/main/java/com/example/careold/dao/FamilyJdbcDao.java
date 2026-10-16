package com.example.careold.dao;

import com.example.careold.domain.Family;
import com.example.careold.domain.FamilyDto;
import com.example.careold.domain.FamilyDto2;
import com.example.careold.common.PasswordUtil;
import org.springframework.jdbc.core.BeanPropertyRowMapper;
import org.springframework.jdbc.support.GeneratedKeyHolder;
import org.springframework.jdbc.support.KeyHolder;
import org.springframework.stereotype.Repository;

import java.sql.PreparedStatement;
import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class FamilyJdbcDao extends JdbcCom implements FamilyDao {
    @Override
    public List<String> notificationPhones() {
        return jdbcTemplate.queryForList("select distinct f.family_phone from family f inner join oldperson o on o.family_id=f.family_id where f.family_phone is not null",String.class);
    }


    @Override
    public PageResult<FamilyDto> findFamily(String familyName, PageRequest page) {
        String value=familyName==null?"":familyName;
        return PagedQuery.query(jdbcTemplate,"SELECT o.old_id,o.name,o.family_id,f.family_name,f.family_relation,f.family_phone,f.family_sex,f.family_address FROM oldperson o INNER JOIN family f ON f.family_id=o.family_id WHERE (?='' OR f.family_name LIKE ?) ORDER BY o.old_id",FamilyDto.class,page,value,"%"+value+"%");
    }

    @Override
    public FamilyDto findFamilyByPhone(String phone) {
        String sql="SELECT old_id,name,oldperson.family_id,family_name,family_relation,family_phone,family_sex,family_address,family_tou_pic from oldperson inner JOIN family on family.family_id=oldperson.family_id where family_phone=?";

        List<FamilyDto> list=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(FamilyDto.class),phone);
        return list.isEmpty()?null:list.get(0);
    }

    @Override
    public Family findByPhoneForLogin(String phone) {
        String sql = "select * from family where family_phone=?";
        List<Family> families = jdbcTemplate.query(sql, new BeanPropertyRowMapper<>(Family.class), phone);
        return families.isEmpty() ? null : families.get(0);
    }

    @Override
    public String findFamilyIdByPhone(String phone) {
        String sql="select  family_id from family where family_phone = ?";
        String familyId=jdbcTemplate.queryForObject(sql,String.class,phone);
        return familyId;
    }

    @Override
    public FamilyDto2 getFamilyByPhoneForFamily(String phone) {
        String sql="select family_id,family_name,family_sex,family_address,family_tou_pic from family where family_phone=?";
        List<FamilyDto2> families=jdbcTemplate.query(sql,new BeanPropertyRowMapper<>(FamilyDto2.class),phone);
        return families.isEmpty()?null:families.get(0);
    }

    @Override
    public int updateFamilyByPhoneForFamily(FamilyDto2 familyDto2) {
        String sql="update family set family_name=?,family_sex=?,family_address=?,family_phone=? where family_id=?";
        return jdbcTemplate.update(sql,familyDto2.getFamilyName(),familyDto2.getFamilySex(),familyDto2.getFamilyAddress(),familyDto2.getFamilyPhone(),familyDto2.getFamilyId());
    }

    @Override
    public int updateFamilyPasswordByPhoneForFamily(String password,String phone) {
        String sql="update family set family_password=? where family_phone=?";
        return jdbcTemplate.update(sql,PasswordUtil.hash(password),phone);
    }

    @Override
    public int updateFamilyTouPic(String pic,String phone) {
        String sql="update family set family_tou_pic=?  where family_phone=?";
         return jdbcTemplate.update(sql,pic,phone);
    }

    @Override
    public Integer findOldIdByPhone(String phone) {
        return jdbcTemplate.queryForObject("select old_id from oldperson where phone=?", Integer.class, phone);
    }

    @Override
    public int save(Family family) {
        String sql="insert into family(family_name,family_relation,family_phone,family_sex,family_address) VALUES(?,?,?,?,?) ";
        return jdbcTemplate.update(sql,family.getFamilyName(),family.getFamilyRelation(),family.getFamilyPhone(),family.getFamilySex(),family.getFamilyAddress());
    }

    @Override
    public Integer saveAndGetId(Family family) {
        String sql = "insert into family(family_name,family_relation,family_phone,family_sex,family_address,family_password)"
                + " VALUES(?,?,?,?,?,?)";
        KeyHolder keyHolder = new GeneratedKeyHolder();
        int rows = jdbcTemplate.update(connection -> {
            PreparedStatement ps = connection.prepareStatement(sql, new String[]{"family_id"});
            ps.setString(1, family.getFamilyName());
            ps.setString(2, family.getFamilyRelation());
            ps.setString(3, family.getFamilyPhone());
            ps.setString(4, family.getFamilySex());
            ps.setString(5, family.getFamilyAddress());
            ps.setString(6, PasswordUtil.hash(family.getFamilyPassword()));
            return ps;
        }, keyHolder);
        if (rows != 1 || keyHolder.getKey() == null) {
            return null;
        }
        return keyHolder.getKey().intValue();
    }

    @Override
    public int countByPhone(String phone) {
        String sql = "select count(1) from family where family_phone = ?";
        Integer count = jdbcTemplate.queryForObject(sql, Integer.class, phone);
        return count == null ? 0 : count;
    }

    @Override
    public int update(Family family) {        String sql="UPDATE family SET family_name=?,family_relation=?,family_phone=?,family_sex=?,family_address=? WHERE family_id=?";
        return jdbcTemplate.update(sql,family.getFamilyName(),family.getFamilyRelation(),family.getFamilyPhone(),family.getFamilySex(),family.getFamilyAddress(),family.getFamilyId());
    }


    @Override
    public int delete(String s) {
        jdbcTemplate.update("update oldperson set family_id=null where family_id=?",s);
        String sql="DELETE FROM family WHERE family_id=?";
        return jdbcTemplate.update(sql,s);
    }
}

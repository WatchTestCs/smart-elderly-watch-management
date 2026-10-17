package com.example.careold.dao;

import com.example.careold.domain.InviteCode;
import com.example.careold.domain.InviteCodeDto;


public interface InviteCodeDao {

    com.example.careold.domain.PageResult<InviteCodeDto> findInviteCodes(String state, com.example.careold.domain.PageRequest page);

    /** 按邀请码查询（不存在返回 null） */
    InviteCodeDto findByCode(String code);

    InviteCodeDto findByCodeForUpdate(String code);

    /** 生成一个数据库中不存在的邀请码 */
    String generateCode();

    /** 插入邀请码，返回自增主键，失败返回 null */
    Integer save(InviteCode inviteCode);

    /** 标记邀请码已被使用 */
    int markUsed(Integer id, Integer familyId, String usedTime);

    /** 删除未被使用的邀请码 */
    int deleteUnused(Integer id);

    /** 删除邀请码绑定的老人的记录（老人被删除时清理） */
    int deleteByOldId(Integer oldId);
}

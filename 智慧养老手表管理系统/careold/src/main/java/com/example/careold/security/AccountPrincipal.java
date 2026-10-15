package com.example.careold.security;

import java.io.Serializable;

/** 会话只保存身份信息，不保存数据库实体或密码。 */
public record AccountPrincipal(String role, Integer id, String phone, Integer oldId,
                               String displayName) implements Serializable {
}

package com.example.careold.domain;

import java.util.List;
import java.util.Map;

public record PageResult<T>(List<T> items, long total, int page, int pageSize) {
    public void putInto(Map<String,Object> result, String key) {
        result.put(key,items); result.put("total",total); result.put("page",page); result.put("pageSize",pageSize);
    }
}

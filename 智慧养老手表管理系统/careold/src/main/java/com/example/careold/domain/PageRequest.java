package com.example.careold.domain;

import java.util.Map;
import org.springframework.http.HttpStatus;
import org.springframework.web.server.ResponseStatusException;

public record PageRequest(int page, int pageSize) {
    public PageRequest {
        if (page < 1 || page > 100000 || pageSize < 1 || pageSize > 100)
            throw new ResponseStatusException(HttpStatus.BAD_REQUEST,"分页参数超出允许范围");
    }
    public static PageRequest from(Map<?,?> parameters) {
        return new PageRequest(number(parameters,"page",1),number(parameters,"pageSize",20));
    }
    private static int number(Map<?,?> parameters, String key, int fallback) {
        Object value=parameters==null?null:parameters.get(key);
        if (value==null) return fallback;
        try { return Integer.parseInt(value.toString()); }
        catch (NumberFormatException e) { throw new ResponseStatusException(HttpStatus.BAD_REQUEST,"分页参数必须为整数"); }
    }
    public int offset() { return (page-1)*pageSize; }
}

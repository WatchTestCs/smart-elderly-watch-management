package com.example.careold.security;

import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import java.util.Map;

@RestControllerAdvice
public class ApiErrorHandler {
    @ExceptionHandler(DataIntegrityViolationException.class)
    public ResponseEntity<Map<String, String>> conflictingWrite(DataIntegrityViolationException error) {
        return ResponseEntity.status(409).body(Map.of("returnCode", "0000", "message", "记录已存在或关联状态已变化，请刷新后重试"));
    }
}

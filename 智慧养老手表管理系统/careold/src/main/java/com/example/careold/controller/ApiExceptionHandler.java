package com.example.careold.controller;

import jakarta.servlet.http.HttpServletRequest;
import java.util.Map;
import java.util.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@RestControllerAdvice
public class ApiExceptionHandler {
    private static final Logger LOG = LoggerFactory.getLogger(ApiExceptionHandler.class);

    @ExceptionHandler(MethodArgumentNotValidException.class)
    ResponseEntity<?> validation() {
        // Never include rejected credentials or personal data in the response/log.
        return ResponseEntity.badRequest().body(Map.of("returnCode","0000","message","请求字段不完整或超出允许范围"));
    }

    @ExceptionHandler({DataAccessException.class, IllegalStateException.class})
    ResponseEntity<?> persistence(RuntimeException error, HttpServletRequest request) {
        String reference = UUID.randomUUID().toString();
        LOG.error("请求失败：path={}, reference={}, type={}, location={}",request.getRequestURI(),reference,
                error.getClass().getSimpleName(),error.getStackTrace().length==0?"unknown":error.getStackTrace()[0]);
        return ResponseEntity.internalServerError().body(Map.of("returnCode","0000","message","操作未完成，请重试；如仍失败，请提供此编号以便排查", "reference",reference));
    }
}

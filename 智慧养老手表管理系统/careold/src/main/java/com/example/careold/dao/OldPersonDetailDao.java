package com.example.careold.dao;

import com.example.careold.domain.OldPersonDetail;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface OldPersonDetailDao {
    PageResult<OldPersonDetail> getOldPersonDetail(String name, PageRequest page);
}

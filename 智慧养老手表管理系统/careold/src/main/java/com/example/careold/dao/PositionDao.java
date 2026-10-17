package com.example.careold.dao;

import com.example.careold.domain.PositionDto;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface PositionDao {
    PageResult<PositionDto> getPosition(String name, PageRequest page);

    List<PositionDto> getPositionFamily(int oldId);
}

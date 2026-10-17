package com.example.careold.dao;

import com.example.careold.domain.StepDto;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface StepDao {
    PageResult<StepDto> getStepSleep(String name, String timeFirst, String timeLast, PageRequest page);

    List<StepDto> getStepSleepDetail(int oldId);

    public List<StepDto> getStepSleepFamily(int oldId);

    List<String> getSleepBadCount();
}

package com.example.careold.dao;

import com.example.careold.domain.WatchDto;
import com.example.careold.domain.WatchInfo;

import java.util.List;
import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface WatchInfoDao {

    PageResult<WatchDto> getWatchInfo(String oldName, PageRequest page);

    List<WatchDto> getWatchInfo2(int deviceId);

    int addWatchInfo(WatchInfo watchInfo);

    int updateWatchInfo(WatchInfo watchInfo);

    int deleteWatchInfo(String deviceId);
}

package com.example.careold.dao;

import com.example.careold.domain.Text;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface TextDao {
    PageResult<Text> selectTextAll(String title, PageRequest page);

    Text selectTextDetail(int textId);

    int addText(Text text);

    int deleteText(int id);

    int update(Text text);

    int nextPicId();
}

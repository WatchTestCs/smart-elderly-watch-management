package com.example.careold.dao;

import com.example.careold.domain.Customer;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

public interface CustomerDao {
    PageResult<Customer> getCustomer(String customerName, PageRequest page);

    int addCustomer(Customer customer);

    int updateCustomer(Customer customer);

    int deleteCustomer(int customerId);

}

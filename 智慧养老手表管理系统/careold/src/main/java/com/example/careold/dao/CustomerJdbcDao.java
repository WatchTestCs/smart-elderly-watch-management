package com.example.careold.dao;

import com.example.careold.domain.Customer;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

import com.example.careold.domain.PageRequest;
import com.example.careold.domain.PageResult;

@Repository
public class CustomerJdbcDao implements CustomerDao{

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Override
    public PageResult<Customer> getCustomer(String customerName, PageRequest page) {
        String value=customerName==null?"":customerName;
        return PagedQuery.query(jdbcTemplate,"SELECT * FROM customer WHERE (?='' OR customer_name LIKE ?) ORDER BY customer_id",Customer.class,page,value,"%"+value+"%");
    }

    @Override
    public int addCustomer(Customer customer) {
        String sql="insert into customer(customer_name,customer_age,customer_sex,customer_phone,customer_address) values(?,?,?,?,?)";
        return jdbcTemplate.update(sql,customer.getCustomerName(),customer.getCustomerAge(),customer.getCustomerSex(),customer.getCustomerPhone(),customer.getCustomerAddress());
    }

    @Override
    public int updateCustomer(Customer customer) {
        String sql="update customer set customer_name=?,customer_age=?,customer_sex=?,customer_phone=?,customer_address=? where customer_id=?";
        return jdbcTemplate.update(sql,customer.getCustomerName(),customer.getCustomerAge(),customer.getCustomerSex(),customer.getCustomerPhone(),customer.getCustomerAddress(),customer.getCustomerId());
    }

    @Override
    public int deleteCustomer(int customerId) {
        String sql="delete from customer where customer_id=?";
        return jdbcTemplate.update(sql,customerId);
    }

}

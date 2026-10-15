
package com.example.careold.common;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;
public class DateConst {
	private DateConst(){}
	public static final String DATE_MODEL_1 = "yyyy-MM-dd HH:mm:ss.SSS";
	public static final String DATE_MODEL_2 = "yyyy-MM-dd HH:mm:ss";
	
	
	public static final String DATE_MODEL_5 = "yyyy-MM-dd";
	

	
	public static final String DATE_MODEL_8 = "yyyyMMddHHmmss";
	public static String dateToString(Date date,String dateModel){
		//可以方便地修改日期格式
		SimpleDateFormat dateFormat = new SimpleDateFormat( dateModel );
		return dateFormat.format( date );
	}

	public static Date toDate(String value, String dateModel) {
		if (value == null) {
			return null;
		}
		SimpleDateFormat dateFormat = new SimpleDateFormat(dateModel);
		dateFormat.setLenient(false);
		try {
			return dateFormat.parse(value);
		} catch (ParseException e) {
			return null;
		}
	}
}

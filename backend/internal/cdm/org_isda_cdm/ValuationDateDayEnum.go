/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines the conditions for the day for a Valuation Date.
   */
  
  const (
  /**
   * The Date can be a given day on the regular calendar.
   */
  ValuationDateDayEnum_CALENDAR_DAY ValuationDateDayEnum = iota + 1
  /**
   * Friday
   */
  ValuationDateDayEnum_FRI ValuationDateDayEnum = iota + 1
  /**
   * The Date must be a day on which commercial banks are open for general business in the local market.
   */
  ValuationDateDayEnum_LOCAL_BUSINESS_DAY ValuationDateDayEnum = iota + 1
  /**
   * Monday
   */
  ValuationDateDayEnum_MON ValuationDateDayEnum = iota + 1
  /**
   * The Date must be a New York Banking Day, that is a day, other than a Saturday or Sunday, on which banks are open for general commercial business in New York, USA.
   */
  ValuationDateDayEnum_NEW_YORK_BANKING_DAY ValuationDateDayEnum = iota + 1
  /**
   * Saturday
   */
  ValuationDateDayEnum_SAT ValuationDateDayEnum = iota + 1
  /**
   * Sunday
   */
  ValuationDateDayEnum_SUN ValuationDateDayEnum = iota + 1
  /**
   * Thursday
   */
  ValuationDateDayEnum_THU ValuationDateDayEnum = iota + 1
  /**
   * Tuesday
   */
  ValuationDateDayEnum_TUE ValuationDateDayEnum = iota + 1
  /**
   * Wednesday
   */
  ValuationDateDayEnum_WED ValuationDateDayEnum = iota + 1
  )    

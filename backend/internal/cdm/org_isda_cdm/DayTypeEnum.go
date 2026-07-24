/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Lists the enumerated values to specify the day type classification used in counting the number of days between two dates.
   */
  
  const (
  /**
   * Applies when calculating the number of days between two dates the count includes only business days.
   */
  DayTypeEnum_BUSINESS DayTypeEnum = iota + 1
  /**
   * Applies when calculating the number of days between two dates the count includes all calendar days.
   */
  DayTypeEnum_CALENDAR DayTypeEnum = iota + 1
  /**
   * Applies when calculating the number of days between two dates the count includes only currency business days.
   */
  DayTypeEnum_CURRENCY_BUSINESS DayTypeEnum = iota + 1
  /**
   * Applies when calculating the number of days between two dates the count includes only stock exchange business days.
   */
  DayTypeEnum_EXCHANGE_BUSINESS DayTypeEnum = iota + 1
  /**
   * Applies when calculating the number of days between two dates the count includes only scheduled trading days.
   */
  DayTypeEnum_SCHEDULED_TRADING_DAY DayTypeEnum = iota + 1
  )    

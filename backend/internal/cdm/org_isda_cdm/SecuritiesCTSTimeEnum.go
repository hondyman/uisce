/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Details the day on which securities collateral is required to be transferred relative to the Notification Time.
   */
  
  const (
  /**
   * The securities collateral should be transferred on the first local business day.
   */
  SecuritiesCTSTimeEnum_FIRST_DAY SecuritiesCTSTimeEnum = iota + 1
  /**
   * The securities collateral should be transferred on the next day.
   */
  SecuritiesCTSTimeEnum_NEXT SecuritiesCTSTimeEnum = iota + 1
  /**
   * Exception value.
   */
  SecuritiesCTSTimeEnum_OTHER SecuritiesCTSTimeEnum = iota + 1
  /**
   * The securities collateral should be transferred on the same day.
   */
  SecuritiesCTSTimeEnum_SAME SecuritiesCTSTimeEnum = iota + 1
  /**
   * The securities collateral should be transferred on the second local business day.
   */
  SecuritiesCTSTimeEnum_SECOND_DAY SecuritiesCTSTimeEnum = iota + 1
  /**
   * The securities collateral should be transferred on the third local business day.
   */
  SecuritiesCTSTimeEnum_THIRD_DAY SecuritiesCTSTimeEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Details the day on which cash collateral is required to be transferred relative to the Notification Time.
   */
  
  const (
  /**
   * The cash collateral should be transferred on the first local business day.
   */
  CashCTSTimeEnum_FIRST_LBD CashCTSTimeEnum = iota + 1
  /**
   * The cash collateral should be transferred on the next day.
   */
  CashCTSTimeEnum_NEXT CashCTSTimeEnum = iota + 1
  /**
   * Exception value.
   */
  CashCTSTimeEnum_OTHER CashCTSTimeEnum = iota + 1
  /**
   * The cash collateral should be transferred on the same day.
   */
  CashCTSTimeEnum_SAME CashCTSTimeEnum = iota + 1
  /**
   * The cash collateral should be transferred on the second local business day.
   */
  CashCTSTimeEnum_SECOND_LBD CashCTSTimeEnum = iota + 1
  )    

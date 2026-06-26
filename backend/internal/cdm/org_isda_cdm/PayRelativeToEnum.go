/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify whether payments occur relative to the calculation period start date or end date, each reset date, valuation date or the last pricing date.
   */
  
  const (
  /**
   * Payments will occur relative to the last day of each calculation period.
   */
  PayRelativeToEnum_CALCULATION_PERIOD_END_DATE PayRelativeToEnum = iota + 1
  /**
   * Payments will occur relative to the first day of each calculation period.
   */
  PayRelativeToEnum_CALCULATION_PERIOD_START_DATE PayRelativeToEnum = iota + 1
  /**
   * Payments will occur relative to the last Pricing Date of each Calculation Period.
   */
  PayRelativeToEnum_LAST_PRICING_DATE PayRelativeToEnum = iota + 1
  /**
   * Payments will occur relative to the reset date.
   */
  PayRelativeToEnum_RESET_DATE PayRelativeToEnum = iota + 1
  /**
   * Payments will occur relative to the valuation date.
   */
  PayRelativeToEnum_VALUATION_DATE PayRelativeToEnum = iota + 1
  )    

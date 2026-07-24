/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Details of how cash collateral is valued when resolving disputes.
   */
  
  const (
  /**
   * Cash - Amount thereof.
   */
  ValueCashEnum_CASH_AMOUNT ValueCashEnum = iota + 1
  /**
   * Cash - amount thereof multiplied by Valuation Percentage.
   */
  ValueCashEnum_CASH_PERCENTAGE ValueCashEnum = iota + 1
  /**
   * Cash - Amount Thereof multiplied by (Valuation Percentage - FX Haircut).
   */
  ValueCashEnum_CASH_PERCENTAGE_LESS_HAIRCUT ValueCashEnum = iota + 1
  /**
   * Exception value.
   */
  ValueCashEnum_OTHER ValueCashEnum = iota + 1
  )    

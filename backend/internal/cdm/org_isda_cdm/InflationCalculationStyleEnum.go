/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Indicates the style of how the inflation index calculates the payment (e.g. YearOnYear, ZeroCoupon).
   */
  
  const (
  /**
   * YearOnYear means Inflation Index Base is the inflation index for Reference Month that is 12 months prior to Inflation Index Final (subject to interpolation). Inflation Index Base is cashflow dependent.
   */
  InflationCalculationStyleEnum_YEAR_ON_YEAR InflationCalculationStyleEnum = iota + 1
  /**
   * ZeroCoupon means Inflation Index Base used in the CalculationMethod is the inflation index for the Reference Month that is the lag number of months prior to Effective Date in the case of a swap, or Bond Interest Accrual Date in the case of an Asset Swap (subject to interpolation). Inflation Index Base has the same value for each inflation cashflow and Principal Exchange calculation within the trade.
   */
  InflationCalculationStyleEnum_ZERO_COUPON InflationCalculationStyleEnum = iota + 1
  )    

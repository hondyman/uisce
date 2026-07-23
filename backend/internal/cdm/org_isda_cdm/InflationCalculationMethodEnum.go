/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Indicates how to use the inflation index to calculate the payment (e.g. Ratio, Return, Spread). Added for Inflation Asset Swap
   */
  
  const (
  /**
   * (Inflation Index Final / Inflation Index Base). Inflation Index Final is inflation index for Reference Month that is the Lag number of months prior to Payment Date (subject to interpolation). Inflation Index Base subject to the Calculation Style. Used in inflation asset swaps to calculate the inflation coupons and principal exchange.
   */
  InflationCalculationMethodEnum_RATIO InflationCalculationMethodEnum = iota + 1
  /**
   * (Inflation Index Final / Inflation Index Base -1). Inflation Index Final is the inflation index for Reference Month that is the Lag number of months prior to Payment Date (subject to interp). Inflation Index Base subject to the Calculation Style. Used in market standard ZC Inflation swaps.
   */
  InflationCalculationMethodEnum_RETURN InflationCalculationMethodEnum = iota + 1
  /**
   * Inflation Index Final - Inflation Index Base). Inflation Index Final is Index for Ref month the Lag months prior to Payment Date (subject to interp). Inflation Index Base subject to the Calculation Style. Typically used for fixing locks.
   */
  InflationCalculationMethodEnum_SPREAD InflationCalculationMethodEnum = iota + 1
  )    

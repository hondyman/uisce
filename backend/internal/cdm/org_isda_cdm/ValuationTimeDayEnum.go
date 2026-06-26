/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The day at which the value of the collateral and obligations to transfer or return collateral are to be calculated relative to the Valuation Date.
   */
  
  const (
  /**
   * Should be calculated relative to the previous local business day as selected.
   */
  ValuationTimeDayEnum_AS_SELECTED ValuationTimeDayEnum = iota + 1
  /**
   * Should be calculated relative to the previous local business day.
   */
  ValuationTimeDayEnum_PREVIOUS_LOCAL_BUSINESS_DAY ValuationTimeDayEnum = iota + 1
  /**
   * Should be calculated relative to the previous local business day in the city of the Valuation Agent.
   */
  ValuationTimeDayEnum_PREVIOUS_LOCAL_BUSINESS_DAY_CITY_OF_AGENT ValuationTimeDayEnum = iota + 1
  /**
   * Should be calculated relative to the previous local business day in the Relevant Market).
   */
  ValuationTimeDayEnum_PREVIOUS_LOCAL_BUSINESS_DAY_MARKET ValuationTimeDayEnum = iota + 1
  /**
   * Should be calculated on the Valuation Date.
   */
  ValuationTimeDayEnum_VALUATION_DATE ValuationTimeDayEnum = iota + 1
  )    

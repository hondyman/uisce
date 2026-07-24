/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The type of time specified for the Valuation Time.
   */
  
  const (
  /**
   * The valuation time should be as selected.
   */
  ValuationTimeEnum_AS_SELECTED ValuationTimeEnum = iota + 1
  /**
   * The valuation time should be at close of business.
   */
  ValuationTimeEnum_CLOSE_OF_BUSINESS ValuationTimeEnum = iota + 1
  /**
   * The valuation time should be at a specific time.
   */
  ValuationTimeEnum_SPECIFIC_TIME ValuationTimeEnum = iota + 1
  )    

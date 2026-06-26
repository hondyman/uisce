/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the method of calculating payment obligations when a floating rate is negative (either due to a quoted negative floating rate or by operation of a spread that is subtracted from the floating rate).
   */
  
  const (
  /**
   * Negative Interest Rate Method. Per 2000 ISDA Definitions, Section 6.4 Negative Interest Rates, paragraphs (b) and (c).
   */
  NegativeInterestRateTreatmentEnum_NEGATIVE_INTEREST_RATE_METHOD NegativeInterestRateTreatmentEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions section 6.8.6
   */
  NegativeInterestRateTreatmentEnum_ZERO_INTEREST_RATE_EXCLUDING_SPREAD_METHOD NegativeInterestRateTreatmentEnum = iota + 1
  /**
   * Zero Interest Rate Method. Per 2000 ISDA Definitions, Section 6.4. Negative Interest Rates, paragraphs (d) and (e).
   */
  NegativeInterestRateTreatmentEnum_ZERO_INTEREST_RATE_METHOD NegativeInterestRateTreatmentEnum = iota + 1
  )    

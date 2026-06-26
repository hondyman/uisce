/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list that specifies the general rule for periodic interest rate payment.
   */
  
  const (
  /**
   * Denotes payment calculated with reference to a fixed interest rate.
   */
  DebtInterestEnum_FIXED DebtInterestEnum = iota + 1
  /**
   * Denotes payment calculated with reference to a floating interest rate.
   */
  DebtInterestEnum_FLOATING DebtInterestEnum = iota + 1
  /**
   * Denotes payment calculated with reference to one or more price or other indices (other than inflation rates).
   */
  DebtInterestEnum_INDEX_LINKED DebtInterestEnum = iota + 1
  /**
   * Denotes payment calculated with reference to one or more specified inflation rates.
   */
  DebtInterestEnum_INFLATION_LINKED DebtInterestEnum = iota + 1
  /**
   * Denotes a stripped bond representing only the interest component.
   */
  DebtInterestEnum_INTEREST_ONLY DebtInterestEnum = iota + 1
  /**
   * Denotes payment calculated with reference to the inverse of a floating interest rate.
   */
  DebtInterestEnum_INVERSE_FLOATING DebtInterestEnum = iota + 1
  /**
   * Denotes payment calculated with reference to other underlyings (not being floating interest rates, inflation rates or indices) or with a non-linear relationship to floating interest rates, inflation rates or indices.
   */
  DebtInterestEnum_OTHER_STRUCTURED DebtInterestEnum = iota + 1
  /**
   * Denotes a zero coupon bond that does not pay intetrest.
   */
  DebtInterestEnum_ZERO_COUPON DebtInterestEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies how multiple credit ratings are compared when determining the applicable Independent Amount, and whether that measure is highest, lowest, or a comparison of the ratings.
   */
  
  const (
  /**
   * Denotes the average credit notation if several notations are listed.
   */
  IndependentAmountCompareEnum_AVERAGE IndependentAmountCompareEnum = iota + 1
  /**
   * Represents that the credit ratings across multiple credit rating agencies will be compared against one another.
   */
  IndependentAmountCompareEnum_COMPARE IndependentAmountCompareEnum = iota + 1
  /**
   * Denotes the highest credit notation if several notations are listed.
   */
  IndependentAmountCompareEnum_HIGHEST IndependentAmountCompareEnum = iota + 1
  /**
   * Denotes the lowest credit notation if several notations are listed.
   */
  IndependentAmountCompareEnum_LOWEST IndependentAmountCompareEnum = iota + 1
  /**
   * Utilised where bespoke language represents the label characteristics of the rating.
   */
  IndependentAmountCompareEnum_OTHER IndependentAmountCompareEnum = iota + 1
  /**
   * Denotes that a credit notation issued from a defined reference agency is used if several notations are listed.
   */
  IndependentAmountCompareEnum_REFERENCE_AGENCY IndependentAmountCompareEnum = iota + 1
  /**
   * Denotes the second best credit notation if several notations are listed.
   */
  IndependentAmountCompareEnum_SECOND_BEST IndependentAmountCompareEnum = iota + 1
  )    

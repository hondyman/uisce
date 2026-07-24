/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Top level ISDA FRO category.
   */
  
  const (
  /**
   * The rate is calculated by the calculation agents from multiple observations.
   */
  FloatingRateIndexCategoryEnum_CALCULATED FloatingRateIndexCategoryEnum = iota + 1
  /**
   * The rate is obtained by polling several other banks.
   */
  FloatingRateIndexCategoryEnum_REFERENCE_BANKS FloatingRateIndexCategoryEnum = iota + 1
  /**
   * The rate is observed directly from a screen.
   */
  FloatingRateIndexCategoryEnum_SCREEN_RATE FloatingRateIndexCategoryEnum = iota + 1
  )    

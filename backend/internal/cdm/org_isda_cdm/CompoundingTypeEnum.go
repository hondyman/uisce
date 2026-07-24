/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify how the compounding calculation is done
   */
  
  const (
  /**
   * Compounding is done only on business days, i.e. not compounded each day on weekends or holidays.
   */
  CompoundingTypeEnum_BUSINESS CompoundingTypeEnum = iota + 1
  /**
   * Compounding is done on each calendar day.
   */
  CompoundingTypeEnum_CALENDAR CompoundingTypeEnum = iota + 1
  /**
   * Compounding is not applicable
   */
  CompoundingTypeEnum_NONE CompoundingTypeEnum = iota + 1
  )    

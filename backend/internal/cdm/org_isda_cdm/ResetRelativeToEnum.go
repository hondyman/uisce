/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify whether resets occur relative to the first or last day of a calculation period.
   */
  
  const (
  /**
   * Resets occur relative to the last day of a calculation period.
   */
  ResetRelativeToEnum_CALCULATION_PERIOD_END_DATE ResetRelativeToEnum = iota + 1
  /**
   * Resets occur relative to the first day of a calculation period.
   */
  ResetRelativeToEnum_CALCULATION_PERIOD_START_DATE ResetRelativeToEnum = iota + 1
  )    

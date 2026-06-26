/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the rounding direction when rounding of a number to nearest.  Used by function cdm.base.math.RoundToNearest.
   */
  
  const (
  /**
   * A number will be rounded down to the specified nearest number. For example, 529 rounded down to the nearest 10 is 520.
   */
  RoundingModeEnum_DOWN RoundingModeEnum = iota + 1
  /**
   * A number will be rounded up to the specified nearest number. For example, 521 rounded up to the nearest 10 is 530.
   */
  RoundingModeEnum_UP RoundingModeEnum = iota + 1
  )    

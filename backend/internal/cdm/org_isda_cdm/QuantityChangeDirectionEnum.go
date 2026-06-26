/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies whether a quantity change is an increase, a decrease or a replacement, whereby the quantity is always specified as a positive number.
   */
  
  const (
  /**
   * When the quantity should go down by the specified amount.
   */
  QuantityChangeDirectionEnum_DECREASE QuantityChangeDirectionEnum = iota + 1
  /**
   * When the quantity should go up by the specified amount.
   */
  QuantityChangeDirectionEnum_INCREASE QuantityChangeDirectionEnum = iota + 1
  /**
   * When the quantity should be replaced by the specified amount.
   */
  QuantityChangeDirectionEnum_REPLACE QuantityChangeDirectionEnum = iota + 1
  )    

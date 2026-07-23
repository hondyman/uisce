/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the type of averaging used in an Asian option.
   */
  
  const (
  /**
   * The average price is used to derive both the strike and the expiration price.
   */
  AveragingInOutEnum_BOTH AveragingInOutEnum = iota + 1
  /**
   * The average price is used to derive the strike price. Also known as 'Asian strike' style option.
   */
  AveragingInOutEnum_IN AveragingInOutEnum = iota + 1
  /**
   * The average price is used to derive the expiration price. Also known as 'Asian price' style option.
   */
  AveragingInOutEnum_OUT AveragingInOutEnum = iota + 1
  )    

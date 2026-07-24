/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * How is collateral interest to be handled?
   */
  
  const (
  /**
   *  Adjust the collateral balance to include the interest amount 
   */
  CollateralInterestHandlingEnum_ADJUST CollateralInterestHandlingEnum = iota + 1
  /**
   *  Transfer the interest each period 
   */
  CollateralInterestHandlingEnum_TRANSFER CollateralInterestHandlingEnum = iota + 1
  /**
   *  Transfer the interest if it meets certain criteria, or otherwise adjust the collateral balance to reflect the interest amount  
   */
  CollateralInterestHandlingEnum_TRANSFER_OR_ADJUST CollateralInterestHandlingEnum = iota + 1
  )    

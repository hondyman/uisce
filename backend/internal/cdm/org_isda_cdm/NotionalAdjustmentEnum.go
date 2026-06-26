/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the conditions that govern the adjustment to the number of units of the return swap.
   */
  
  const (
  /**
   * The adjustments to the number of units are governed by an execution clause.
   */
  NotionalAdjustmentEnum_EXECUTION NotionalAdjustmentEnum = iota + 1
  /**
   * The adjustments to the number of units are governed by a portfolio rebalancing clause.
   */
  NotionalAdjustmentEnum_PORTFOLIO_REBALANCING NotionalAdjustmentEnum = iota + 1
  /**
   * The adjustments to the number of units are not governed by any specific clause.
   */
  NotionalAdjustmentEnum_STANDARD NotionalAdjustmentEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the time of day which would be considered for valuing the knock event.
   */
  
  const (
  /**
   * At any time during the Knock Determination period (continuous barrier).
   */
  TriggerTimeTypeEnum_ANYTIME TriggerTimeTypeEnum = iota + 1
  /**
   * The close of trading on a day would be considered for valuation.
   */
  TriggerTimeTypeEnum_CLOSING TriggerTimeTypeEnum = iota + 1
  )    

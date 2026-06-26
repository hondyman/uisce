/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify whether an option will trigger or expire depending upon whether the spot rate is above or below the barrier rate.
   */
  
  const (
  /**
   * The underlier price must be equal to the Trigger level.
   */
  TriggerTypeEnum_EQUAL TriggerTypeEnum = iota + 1
  /**
   * The underlier price must be equal to or greater than the Trigger level.
   */
  TriggerTypeEnum_EQUAL_OR_GREATER TriggerTypeEnum = iota + 1
  /**
   * The underlier price must be equal to or less than the Trigger level.
   */
  TriggerTypeEnum_EQUAL_OR_LESS TriggerTypeEnum = iota + 1
  /**
   * The underlier price must be greater than the Trigger level.
   */
  TriggerTypeEnum_GREATER TriggerTypeEnum = iota + 1
  /**
   * The underlier price must be less than the Trigger level.
   */
  TriggerTypeEnum_LESS TriggerTypeEnum = iota + 1
  )    

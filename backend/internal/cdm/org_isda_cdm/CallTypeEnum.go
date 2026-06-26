/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumeration values that indicate the intended status of message type, such as expected call, notification of a call or a margin call.
   */
  
  const (
  /**
   * Identifies an expected Margin Call instruction for either party to notify the other or their service provider of an expected margin call movement.
   */
  CallTypeEnum_EXPECTED_CALL CallTypeEnum = iota + 1
  /**
   * Identifies an actionable Margin Call.
   */
  CallTypeEnum_MARGIN_CALL CallTypeEnum = iota + 1
  /**
   * Identifies a notification of a Margin Call for legal obligation to notify other party to initiate a margin call when notifying party is calculation or valuation agent.
   */
  CallTypeEnum_NOTIFICATION CallTypeEnum = iota + 1
  )    

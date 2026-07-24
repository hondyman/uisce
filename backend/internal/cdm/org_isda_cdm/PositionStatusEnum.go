/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Enumeration to describe the different (risk) states of a Position, whether executed, settled, matured...etc
   */
  
  const (
  /**
   * The position has been cancelled, in case of a cancellation event following an execution.
   */
  PositionStatusEnum_CANCELLED PositionStatusEnum = iota + 1
  /**
   * The position has been closed, in case of a termination event.
   */
  PositionStatusEnum_CLOSED PositionStatusEnum = iota + 1
  /**
   * The position has been executed, which is the point at which risk has been transferred.
   */
  PositionStatusEnum_EXECUTED PositionStatusEnum = iota + 1
  /**
   * Contract has been formed, in case position is on a contractual product.
   */
  PositionStatusEnum_FORMED PositionStatusEnum = iota + 1
  /**
   * The position has settled, in case product is subject to settlement after execution, such as securities.
   */
  PositionStatusEnum_SETTLED PositionStatusEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify what led to the contract or execution closure.
   */
  
  const (
  /**
   * The execution or contract has been allocated.
   */
  ClosedStateEnum_ALLOCATED ClosedStateEnum = iota + 1
  /**
   * The execution or contract has been cancelled.
   */
  ClosedStateEnum_CANCELLED ClosedStateEnum = iota + 1
  /**
   * The (option) contract has been exercised.
   */
  ClosedStateEnum_EXERCISED ClosedStateEnum = iota + 1
  /**
   * The (option) contract has expired without being exercised.
   */
  ClosedStateEnum_EXPIRED ClosedStateEnum = iota + 1
  /**
   * The contract has reached its contractual termination date.
   */
  ClosedStateEnum_MATURED ClosedStateEnum = iota + 1
  /**
   * The contract has been novated. This state applies to the stepped-out contract component of the novation event.
   */
  ClosedStateEnum_NOVATED ClosedStateEnum = iota + 1
  /**
   * The contract has been subject of an early termination event.
   */
  ClosedStateEnum_TERMINATED ClosedStateEnum = iota + 1
  )    

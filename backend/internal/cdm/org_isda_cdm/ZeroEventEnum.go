/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values for defining the relevant trigger(s) for the Minimum Transfer Amount (MTA) to fall to zero.
   */
  
  const (
  /**
   * An Additional Termination Event (ATE).
   */
  ZeroEventEnum_ADDITIONAL_TERMINATION_EVENT ZeroEventEnum = iota + 1
  /**
   * An Event of Default.
   */
  ZeroEventEnum_EVENT_OF_DEFAULT ZeroEventEnum = iota + 1
  /**
   * Utilised where the clause data structure is not able to capture a material aspect of the clause.
   */
  ZeroEventEnum_OTHER ZeroEventEnum = iota + 1
  /**
   * A Potential Event of Default.
   */
  ZeroEventEnum_POTENTIAL_EVENT_OF_DEFAULT ZeroEventEnum = iota + 1
  /**
   * A Termination Event.
   */
  ZeroEventEnum_TERMINATION_EVENT ZeroEventEnum = iota + 1
  /**
   * A Termination Event in respect of which all Transactions are Affected Transactions.
   */
  ZeroEventEnum_TERMINATION_EVENT_ALL_AFFECTED_TRANSACTIONS ZeroEventEnum = iota + 1
  )    

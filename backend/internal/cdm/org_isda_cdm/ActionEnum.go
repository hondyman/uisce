/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to specify the actions associated with transactions.
   */
  
  const (
  /**
   * A cancellation of a prior instance of the transaction event. The eventIdentifier has an associated version greater than 1.
   */
  ActionEnum_CANCEL ActionEnum = iota + 1
  /**
   * A correction of a prior instance of the transaction event. The eventIdentifier has an associated version greater than 1.
   */
  ActionEnum_CORRECT ActionEnum = iota + 1
  /**
   * A new instance of a transaction event, which is also characterized by the fact that the eventIdentifier has an associated version 1.
   */
  ActionEnum_NEW ActionEnum = iota + 1
  )    

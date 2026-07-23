/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Identifies a party to the on-demand repo transaction that has a right to demand for termination of the Security Finance transaction.
   */
  
  const (
  /**
   * As defined in Master Agreement.
   */
  CallingPartyEnum_AS_DEFINED_IN_MASTER_AGREEMENT CallingPartyEnum = iota + 1
  /**
   * Either, Buyer or Seller to the repo transaction.
   */
  CallingPartyEnum_EITHER CallingPartyEnum = iota + 1
  /**
   * Initial buyer to the repo transaction.
   */
  CallingPartyEnum_INITIAL_BUYER CallingPartyEnum = iota + 1
  /**
   * Initial seller to the repo transaction.
   */
  CallingPartyEnum_INITIAL_SELLER CallingPartyEnum = iota + 1
  )    

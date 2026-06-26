/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify an event that has given rise to a fee.
   */
  
  const (
  /**
   * A cash flow resulting from the assignment of a contract to a new counterparty.
   */
  FeeTypeEnum_ASSIGNMENT FeeTypeEnum = iota + 1
  /**
   * The brokerage commission.
   */
  FeeTypeEnum_BROKERAGE_COMMISSION FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with a corporate action
   */
  FeeTypeEnum_CORPORATE_ACTION FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with a credit event.
   */
  FeeTypeEnum_CREDIT_EVENT FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with an increase lifecycle event.
   */
  FeeTypeEnum_INCREASE FeeTypeEnum = iota + 1
  /**
   * The novation fee.
   */
  FeeTypeEnum_NOVATION FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with a partial termination lifecycle event.
   */
  FeeTypeEnum_PARTIAL_TERMINATION FeeTypeEnum = iota + 1
  /**
   * Denotes the amount payable by the buyer to the seller for an option. The premium is paid on the specified premium payment date or on each premium payment date if specified.
   */
  FeeTypeEnum_PREMIUM FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with a renegotiation lifecycle event.
   */
  FeeTypeEnum_RENEGOTIATION FeeTypeEnum = iota + 1
  /**
   * A cash flow associated with a termination lifecycle event.
   */
  FeeTypeEnum_TERMINATION FeeTypeEnum = iota + 1
  /**
   * An upfront cashflow associated to the swap to adjust for a difference between the swap price and the current market price.
   */
  FeeTypeEnum_UPFRONT FeeTypeEnum = iota + 1
  )    

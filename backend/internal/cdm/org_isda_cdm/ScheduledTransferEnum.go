/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The qualification of the type of transfers and cash flows associated with contracts and their lifecycle events.
   */
  
  const (
  /**
   * A cash flow corresponding to a corporate action event.
   */
  ScheduledTransferEnum_CORPORATE_ACTION ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the periodic accrued interests.
   */
  ScheduledTransferEnum_COUPON ScheduledTransferEnum = iota + 1
  /**
   * A cashflow resulting from a credit event.
   */
  ScheduledTransferEnum_CREDIT_EVENT ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the synthetic dividend of an equity underlier asset traded through a derivative instrument.
   */
  ScheduledTransferEnum_DIVIDEND_RETURN ScheduledTransferEnum = iota + 1
  /**
   * A cash flow associated with an exercise lifecycle event.
   */
  ScheduledTransferEnum_EXERCISE ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the return of the fixed interest rate portion of a derivative instrument that has different types of underlying assets, such as a total return swap.
   */
  ScheduledTransferEnum_FIXED_RATE_RETURN ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the return of the floating interest rate portion of a derivative instrument that has different types of underlying assets, such as a total return swap.
   */
  ScheduledTransferEnum_FLOATING_RATE_RETURN ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the compensation for missing assets due to the rounding of digits in the original number of assets to be delivered as per payout calculation.
   */
  ScheduledTransferEnum_FRACTIONAL_AMOUNT ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to the return of the interest rate portion of a derivative instrument that has different types of underlying assets, such as a total return swap.
   */
  ScheduledTransferEnum_INTEREST_RETURN ScheduledTransferEnum = iota + 1
  /**
   * Net interest across payout components. Applicable to products such as interest rate swaps.
   */
  ScheduledTransferEnum_NET_INTEREST ScheduledTransferEnum = iota + 1
  /**
   * A cash flow corresponding to a performance return.  The settlementOrigin attribute on the Transfer should point to the relevant Payout defining the performance calculation.
   */
  ScheduledTransferEnum_PERFORMANCE ScheduledTransferEnum = iota + 1
  /**
   * An amount which corresponds to the notional of the contract for various business reasons. This could be associated to a cashflow (e.g. a principal payment) or a transfer (e.g. delivery of an asset).
   */
  ScheduledTransferEnum_PRINCIPAL ScheduledTransferEnum = iota + 1
  )    

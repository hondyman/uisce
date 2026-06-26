/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Provides a list of possible sub types of the chosen price type, used in conjunction with the PriceTypeEnum.
   */
  
  const (
  /**
   * Denotes a discount factor expressed as a decimal, e.g. 0.95.
   */
  PriceSubTypeEnum_DISCOUNT PriceSubTypeEnum = iota + 1
  /**
   * A generic term for describing a non-scheduled cashflow that can be associated either with the initial contract, with some later corrections to it (e.g. a correction to the day count fraction that has a cashflow impact) or with some lifecycle events. Fees that are specifically associated with termination and partial termination, increase, amendment, and exercise events are qualified accordingly.
   */
  PriceSubTypeEnum_FEE PriceSubTypeEnum = iota + 1
  /**
   * Denotes the amount payable by the buyer to the seller for an option. The premium is paid on the specified premium payment date or on each premium payment date if specified.
   */
  PriceSubTypeEnum_PREMIUM PriceSubTypeEnum = iota + 1
  /**
   * Denotes any rebate that is attributed to the cashflow. For example, where a lender will provide a rebate on the cash given to them by the borrower as collateral on a securities lending trade. 
   */
  PriceSubTypeEnum_REBATE PriceSubTypeEnum = iota + 1
  )    

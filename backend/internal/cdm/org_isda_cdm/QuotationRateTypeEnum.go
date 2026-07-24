/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the type of quotation rate to be obtained from each cash settlement reference bank.
   */
  
  const (
  /**
   * An ask rate.
   */
  QuotationRateTypeEnum_ASK QuotationRateTypeEnum = iota + 1
  /**
   * A bid rate.
   */
  QuotationRateTypeEnum_BID QuotationRateTypeEnum = iota + 1
  /**
   * If optional early termination is applicable to a swap transaction, the rate, which may be a bid or ask rate, which would result, if seller is in-the-money, in the higher absolute value of the cash settlement amount, or, is seller is out-of-the-money, in the lower absolute value of the cash settlement amount.
   */
  QuotationRateTypeEnum_EXERCISING_PARTY_PAYS QuotationRateTypeEnum = iota + 1
  /**
   * A mid-market rate.
   */
  QuotationRateTypeEnum_MID QuotationRateTypeEnum = iota + 1
  )    

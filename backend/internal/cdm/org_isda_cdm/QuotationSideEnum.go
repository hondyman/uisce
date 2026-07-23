/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the side from which perspective a value is quoted.
   */
  
  const (
  /**
   * Denotes a value as the Afternoon fixing reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_AFTERNOON QuotationSideEnum = iota + 1
  /**
   * Denotes a value 'asked' by a seller for an asset, i.e. the value at which a seller is willing to sell.
   */
  QuotationSideEnum_ASK QuotationSideEnum = iota + 1
  /**
   * Denotes a value 'bid' by a buyer for an asset, i.e. the value a buyer is willing to pay.
   */
  QuotationSideEnum_BID QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Closing price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_CLOSING QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the High price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_HIGH QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Index price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_INDEX QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Locational Marginal price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_LOCATIONAL_MARGINAL QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Low price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_LOW QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Marginal Hourly price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MARGINAL_HOURLY QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Market Clearing price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MARKET_CLEARING QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Average of the Bid and Ask prices reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MEAN_OF_BID_AND_ASK QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Average of the High and Low prices reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MEAN_OF_HIGH_AND_LOW QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Average of the Midpoint of prices reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MID QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Morning fixing reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_MORNING QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the National Single price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_NATIONAL_SINGLE QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Official Settlement Price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_OSP QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Official price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_OFFICIAL QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Opening price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_OPENING QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Settlement price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_SETTLEMENT QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Spot price reported in or by the relevant Price Source as specified in the relevant Confirmation.
   */
  QuotationSideEnum_SPOT QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Non-volume Weighted Average of prices effective on the Pricing Date.
   */
  QuotationSideEnum_UN_WEIGHTED_AVERAGE QuotationSideEnum = iota + 1
  /**
   * Denotes a value as the Volume Weighted Average of prices effective on the Pricing Date.
   */
  QuotationSideEnum_WEIGHTED_AVERAGE QuotationSideEnum = iota + 1
  )    

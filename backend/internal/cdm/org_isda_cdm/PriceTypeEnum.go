/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Provides enumerated values for types of prices in the Price data type in order to explain how to interpret the amount and use it in calculations.
   */
  
  const (
  /**
   * Denotes a price expressed as a cash amount in a given currency to purchase a unit of an asset (e.g. a security or a commodity).
   */
  PriceTypeEnum_ASSET_PRICE PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as a cash amount for an upfront fee or other purposes. For example, {amount, unitOfAmount, PerUnitOfAmount} = [12,500, USD, null] = USD 12,500.
   */
  PriceTypeEnum_CASH_PRICE PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as the weighted average of all pairwise correlation coefficients.
   */
  PriceTypeEnum_CORRELATION PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as the dividend payment from a index or share.
   */
  PriceTypeEnum_DIVIDEND PriceTypeEnum = iota + 1
  /**
   * Denotes a rate to convert one currency or other measure of value to another. Foreign Exchange rates are represented in decimals, e.g. {amount, unitOfAmount, PerUnitOfAmount} = [1.23, USD, GBP] = USD 1.23 for every 1 GBP.
   */
  PriceTypeEnum_EXCHANGE_RATE PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as a rate to be applied to quantity/notional amount and represented as decimal, e.g. {amount, unitOfAmount, PerUnitOfAmount} = [0.08, EUR, EUR] = 8%  of the EUR notional quantity/amount or 8 cents for every EUR of notional amount.
   */
  PriceTypeEnum_INTEREST_RATE PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as the the arithmetic average of the squared differences from the mean value of an observable price.
   */
  PriceTypeEnum_VARIANCE PriceTypeEnum = iota + 1
  /**
   * Denotes a price expressed as the the square root of the arithmetic average of the squared differences from the mean value of an observable price.
   */
  PriceTypeEnum_VOLATILITY PriceTypeEnum = iota + 1
  )    

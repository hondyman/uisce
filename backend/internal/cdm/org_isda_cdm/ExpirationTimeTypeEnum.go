/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The time of day at which the equity option expires, for example the official closing time of the exchange.
   */
  
  const (
  /**
   * The time is determined as provided in the relevant Master Confirmation.
   */
  ExpirationTimeTypeEnum_AS_SPECIFIED_IN_MASTER_CONFIRMATION ExpirationTimeTypeEnum = iota + 1
  /**
   * The official closing time of the exchange on the valuation date.
   */
  ExpirationTimeTypeEnum_CLOSE ExpirationTimeTypeEnum = iota + 1
  /**
   * The official closing time of the derivatives exchange on which a derivative contract is listed on that security underlyer.
   */
  ExpirationTimeTypeEnum_DERIVATIVES_CLOSE ExpirationTimeTypeEnum = iota + 1
  /**
   * The time at which the official settlement price is determined.
   */
  ExpirationTimeTypeEnum_OSP ExpirationTimeTypeEnum = iota + 1
  /**
   * The official opening time of the exchange on the valuation date.
   */
  ExpirationTimeTypeEnum_OPEN ExpirationTimeTypeEnum = iota + 1
  /**
   * The time specified in the element equityExpirationTime or valuationTime (as appropriate)
   */
  ExpirationTimeTypeEnum_SPECIFIC_TIME ExpirationTimeTypeEnum = iota + 1
  /**
   * The time at which the official settlement price (following the auction by the exchange) is determined by the exchange.
   */
  ExpirationTimeTypeEnum_XETRA ExpirationTimeTypeEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify points in the day when option exercise and valuation can occur.
   */
  
  const (
  /**
   * The time is determined as provided in the relevant Master Confirmation.
   */
  TimeTypeEnum_AS_SPECIFIED_IN_MASTER_CONFIRMATION TimeTypeEnum = iota + 1
  /**
   * The official closing time of the exchange on the valuation date.
   */
  TimeTypeEnum_CLOSE TimeTypeEnum = iota + 1
  /**
   * The official closing time of the derivatives exchange on which a derivative contract is listed on that security underlier.
   */
  TimeTypeEnum_DERIVATIVES_CLOSE TimeTypeEnum = iota + 1
  /**
   * The time at which the official settlement price is determined.
   */
  TimeTypeEnum_OSP TimeTypeEnum = iota + 1
  /**
   * The official opening time of the exchange on the valuation date.
   */
  TimeTypeEnum_OPEN TimeTypeEnum = iota + 1
  /**
   * The time specified in the element equityExpirationTime or valuationTime (as appropriate).
   */
  TimeTypeEnum_SPECIFIC_TIME TimeTypeEnum = iota + 1
  /**
   * The time at which the official settlement price (following the auction by the exchange) is determined by the exchange.
   */
  TimeTypeEnum_XETRA TimeTypeEnum = iota + 1
  )    

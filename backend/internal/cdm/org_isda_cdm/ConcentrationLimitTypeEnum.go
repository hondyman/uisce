/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumerated values to identify where a concentration limit is applied in the eligible collateral schedule.
   */
  
  const (
  /**
   * Specifies a limit on a single asset in the eligible collateral schedule
   */
  ConcentrationLimitTypeEnum_ASSET ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on all cash valued in the base currency of the eligible collateral schedule.
   */
  ConcentrationLimitTypeEnum_BASE_CURRENCY_EQUIVALENT ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on a single industry sector in the eligible collateral schedule.
   */
  ConcentrationLimitTypeEnum_INDUSTRY_SECTOR ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit of the issue compared to the outstanding amount of the asset on the market.
   */
  ConcentrationLimitTypeEnum_ISSUE_OUTSTANDING_AMOUNT ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on a single issuer in the eligible collateral schedule.
   */
  ConcentrationLimitTypeEnum_ISSUER ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit of the issue calculated as a percentage of the market capitalisation of the asset on the market.
   */
  ConcentrationLimitTypeEnum_MARKET_CAPITALISATION ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on the total outstanding balance for an asset in the portfolio.
   */
  ConcentrationLimitTypeEnum_OUTSTANDING_BALANCE ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on a single exchange in the eligible collateral schedule.
   */
  ConcentrationLimitTypeEnum_PRIMARY_EXCHANGE ConcentrationLimitTypeEnum = iota + 1
  /**
   * Specifies a limit on a single issuer in the eligible collateral schedule at the ultimate parent institution level.
   */
  ConcentrationLimitTypeEnum_ULTIMATE_PARENT_INSTITUTION ConcentrationLimitTypeEnum = iota + 1
  )    

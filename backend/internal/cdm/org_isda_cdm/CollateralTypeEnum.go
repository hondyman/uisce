/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the types of collateral that are accepted by the Lender
   */
  
  const (
  /**
   * Security Lending Trades against Cash collateral
   */
  CollateralTypeEnum_CASH CollateralTypeEnum = iota + 1
  /**
   * Security Lending Trades against CashPool collateral
   */
  CollateralTypeEnum_CASH_POOL CollateralTypeEnum = iota + 1
  /**
   * Security Lending Trades against NonCash collateral
   */
  CollateralTypeEnum_NON_CASH CollateralTypeEnum = iota + 1
  )    

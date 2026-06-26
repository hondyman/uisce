/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Details of how securities collateral is valued when resolving disputes.
   */
  
  const (
  /**
   * Securities collateral is valued using the bid price multiplied by the valuation percentage.
   */
  ValueSecuritiesEnum_BID_PRICE_PERCENTAGE ValueSecuritiesEnum = iota + 1
  /**
   * Securities collateral is valued using the bid price multiplied by the valuation percentage less the FX haircut.
   */
  ValueSecuritiesEnum_BID_PRICE_PERCENTAGE_LESS_HAIRCUT ValueSecuritiesEnum = iota + 1
  /**
   * Securities collateral is valued using the mid price multiplied by the valuation percentage.
   */
  ValueSecuritiesEnum_MID_PRICE_PERCENTAGE ValueSecuritiesEnum = iota + 1
  /**
   * Securities collateral is valued using the mid price multiplied by the valuation percentage less the FX haircut.
   */
  ValueSecuritiesEnum_MID_PRICE_PERCENTAGE_LESS_HAIRCUT ValueSecuritiesEnum = iota + 1
  /**
   * Exception value.
   */
  ValueSecuritiesEnum_OTHER ValueSecuritiesEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * If there is an alternative to interest amounts, how is it specified?
   */
  
  const (
  /**
   * The standard calculation of the Interest Amount is replaced with the amount of interest the secured party actually receives in relation to the Cash collateral.
   */
  AlternativeToInterestAmountEnum_ACTUAL_AMOUNT_RECEIVED AlternativeToInterestAmountEnum = iota + 1
  /**
   * An other alternative option outside these choices that can be described as an alternative provision.
   */
  AlternativeToInterestAmountEnum_OTHER AlternativeToInterestAmountEnum = iota + 1
  /**
   * Interest amount is not transferred if transfer would create or increase a delivery amount.
   */
  AlternativeToInterestAmountEnum_STANDARD AlternativeToInterestAmountEnum = iota + 1
  /**
   * Interest amount is not transferred if transfer would create or increase a delivery amount. (This is the 'Standard' provision). However, interest Amount will be transferred if Delivery Amount is below Minimum Transfer Amount.
   */
  AlternativeToInterestAmountEnum_TRANSFER_IF_DELIVERY_AMOUNT_BELOW_MTA AlternativeToInterestAmountEnum = iota + 1
  )    

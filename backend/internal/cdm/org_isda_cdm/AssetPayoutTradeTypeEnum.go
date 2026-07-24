/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * An enumerator to differentiate the different trade types used in securities finance and modelled on an AssetPayout.
   */
  
  const (
  /**
   * In the case of a buy/sell-back, there is no income payment between buyer and seller. Instead, the repurchase price to be paid on the repurchase date is reduced by the amount of the income payment on the collateral plus some extra interest to compensate the seller for the delay between the income payment date on the collateral and the repurchase date of the repo.
   */
  AssetPayoutTradeTypeEnum_BUY_SELL_BACK AssetPayoutTradeTypeEnum = iota + 1
  /**
   * In the case of a repurchase transaction, an immediate and equal income payment (often call a manufactured payment) is made by the buyer to the seller.
   */
  AssetPayoutTradeTypeEnum_REPO AssetPayoutTradeTypeEnum = iota + 1
  )    

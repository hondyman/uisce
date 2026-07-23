/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the consequences of extraordinary events relating to the underlying.
   */
  
  const (
  /**
   * The trade continues such that the underlying now consists of the New Shares and/or the Other Consideration, if any, and the proceeds of any redemption, if any, that the holder of the underlying Shares would have been entitled to.
   */
  ShareExtraordinaryEventEnum_ALTERNATIVE_OBLIGATION ShareExtraordinaryEventEnum = iota + 1
  /**
   * The Calculation Agent will determine what adjustment is required to offset any change to the economics of the trade. If the Calculation Agent cannot achieve this, the trade goes to Cancellation and Payment with the Calculation Agent deciding on the value of the cancellation fee. Adjustments may not be made to account solely for changes in volatility, expected dividends, stock loan rate or liquidity.
   */
  ShareExtraordinaryEventEnum_CALCULATION_AGENT ShareExtraordinaryEventEnum = iota + 1
  /**
   * The trade is cancelled and a cancellation fee will be paid by one party to the other.
   */
  ShareExtraordinaryEventEnum_CANCELLATION_AND_PAYMENT ShareExtraordinaryEventEnum = iota + 1
  /**
   * If this is a Share-for-Combined merger event (Shares are replaced with New Shares and Other Consideration), then different treatment can be applied to each component if the parties have specified this.
   */
  ShareExtraordinaryEventEnum_COMPONENT ShareExtraordinaryEventEnum = iota + 1
  /**
   * The Calculation Agent will determine what adjustment is required to offset any change to the economics of the trade. If the Calculation Agent cannot achieve this, the trade goes to Cancellation and Payment with the Calculation Agent deciding on the value of the cancellation fee. Adjustments to account for changes in volatility, expected dividends, stock loan rate or liquidity are allowed.
   */
  ShareExtraordinaryEventEnum_MODIFIED_CALCULATION_AGENT ShareExtraordinaryEventEnum = iota + 1
  /**
   * The trade will be adjusted by the Calculation Agent in accordance with the adjustments made by any exchange on which options on the underlying are listed.
   */
  ShareExtraordinaryEventEnum_OPTIONS_EXCHANGE ShareExtraordinaryEventEnum = iota + 1
  /**
   * Applies to Basket Transactions. The portion of the Basket made up by the affected Share will be cancelled and a cancellation fee will be paid from one party to the other. The remainder of the trade continues.
   */
  ShareExtraordinaryEventEnum_PARTIAL_CANCELLATION_AND_PAYMENT ShareExtraordinaryEventEnum = iota + 1
  )    

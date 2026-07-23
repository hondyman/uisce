/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify how the composition of Dividends is to be determined.
   */
  
  const (
  /**
   * The Calculation Agent determines the composition of dividends (subject to conditions).
   */
  DividendCompositionEnum_CALCULATION_AGENT_ELECTION DividendCompositionEnum = iota + 1
  /**
   * The Equity Amount Receiver determines the composition of dividends (subject to conditions).
   */
  DividendCompositionEnum_EQUITY_AMOUNT_RECEIVER_ELECTION DividendCompositionEnum = iota + 1
  )    

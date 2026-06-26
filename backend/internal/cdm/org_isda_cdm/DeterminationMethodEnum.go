/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the method according to which an amount or a date is determined.
   */
  
  const (
  /**
   * Agreed separately between the parties.
   */
  DeterminationMethodEnum_AGREED_INITIAL_PRICE DeterminationMethodEnum = iota + 1
  /**
   * As specified in Master Confirmation.
   */
  DeterminationMethodEnum_AS_SPECIFIED_IN_MASTER_CONFIRMATION DeterminationMethodEnum = iota + 1
  /**
   * Determined by the Calculation Agent.
   */
  DeterminationMethodEnum_CALCULATION_AGENT DeterminationMethodEnum = iota + 1
  /**
   * Official Closing Price.
   */
  DeterminationMethodEnum_CLOSING_PRICE DeterminationMethodEnum = iota + 1
  /**
   * Determined by the Currency of Equity Dividends.
   */
  DeterminationMethodEnum_DIVIDEND_CURRENCY DeterminationMethodEnum = iota + 1
  /**
   * The initial Index Level is the level of the Expiring Contract as provided in the Master Confirmation.
   */
  DeterminationMethodEnum_EXPIRING_CONTRACT_LEVEL DeterminationMethodEnum = iota + 1
  /**
   * Determined by the Hedging Party.
   */
  DeterminationMethodEnum_HEDGE_EXECUTION DeterminationMethodEnum = iota + 1
  /**
   * Issuer Payment Currency.
   */
  DeterminationMethodEnum_ISSUER_PAYMENT_CURRENCY DeterminationMethodEnum = iota + 1
  /**
   * Net Asset Value.
   */
  DeterminationMethodEnum_NAV DeterminationMethodEnum = iota + 1
  /**
   * OSP Price.
   */
  DeterminationMethodEnum_OSP_PRICE DeterminationMethodEnum = iota + 1
  /**
   * Opening Price of the Market.
   */
  DeterminationMethodEnum_OPEN_PRICE DeterminationMethodEnum = iota + 1
  /**
   * Settlement Currency.
   */
  DeterminationMethodEnum_SETTLEMENT_CURRENCY DeterminationMethodEnum = iota + 1
  /**
   * Date on which the strike is determined in respect of a forward starting swap.
   */
  DeterminationMethodEnum_STRIKE_DATE_DETERMINATION DeterminationMethodEnum = iota + 1
  /**
   * Official TWAP Price.
   */
  DeterminationMethodEnum_TWAP_PRICE DeterminationMethodEnum = iota + 1
  /**
   * Official VWAP Price.
   */
  DeterminationMethodEnum_VWAP_PRICE DeterminationMethodEnum = iota + 1
  /**
   * Price determined at valuation time.
   */
  DeterminationMethodEnum_VALUATION_TIME DeterminationMethodEnum = iota + 1
  )    

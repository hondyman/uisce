/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Second level ISDA FRO category.
   */
  
  const (
  /**
   * An ISDA-defined calculated rate done using arithmetic averaging.
   */
  FloatingRateIndexStyleEnum_AVERAGE_FRO FloatingRateIndexStyleEnum = iota + 1
  /**
   * An ISDA-defined calculated rate done using arithmetic averaging.
   */
  FloatingRateIndexStyleEnum_COMPOUNDED_FRO FloatingRateIndexStyleEnum = iota + 1
  /**
   * A published index calculated using compounding.
   */
  FloatingRateIndexStyleEnum_COMPOUNDED_INDEX FloatingRateIndexStyleEnum = iota + 1
  /**
   * A published index using a methodology defined by the publisher, e.g. S&P 500.
   */
  FloatingRateIndexStyleEnum_INDEX FloatingRateIndexStyleEnum = iota + 1
  FloatingRateIndexStyleEnum_OTHER FloatingRateIndexStyleEnum = iota + 1
  FloatingRateIndexStyleEnum_OVERNIGHT FloatingRateIndexStyleEnum = iota + 1
  /**
   *  A published rate computed using an averaging methodology.
   */
  FloatingRateIndexStyleEnum_PUBLISHED_AVERAGE FloatingRateIndexStyleEnum = iota + 1
  FloatingRateIndexStyleEnum_SPECIFIED_FORMULA FloatingRateIndexStyleEnum = iota + 1
  /**
   * A rate representing the market rate for swaps of a given maturity.
   */
  FloatingRateIndexStyleEnum_SWAP_RATE FloatingRateIndexStyleEnum = iota + 1
  /**
   * A rate specified over a given term, such as a libor-type rate.
   */
  FloatingRateIndexStyleEnum_TERM_RATE FloatingRateIndexStyleEnum = iota + 1
  )    

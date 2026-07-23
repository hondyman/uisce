/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the period term as part of a periodic schedule, i.e. the calculation period end date within the regular part of the calculation period. The value could be a rule, e.g. IMM Settlement Dates, which is the 3rd Wednesday of the month, or it could be a specific day of the month, such as the first day of the applicable month.
   */
  
  const (
  /**
   * Rolls on month end dates irrespective of the length of the month and the previous roll day.
   */
  RollConventionEnum_EOM RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Friday
   */
  RollConventionEnum_FRI RollConventionEnum = iota + 1
  /**
   * Roll days are determined according to the FRN Convention or Euro-dollar Convention as described in ISDA 2000 definitions.
   */
  RollConventionEnum_FRN RollConventionEnum = iota + 1
  /**
   * IMM Settlement Dates. The third Wednesday of the (delivery) month.
   */
  RollConventionEnum_IMM RollConventionEnum = iota + 1
  /**
   * The last trading day of the Sydney Futures Exchange 90 Day Bank Accepted Bills Futures contract (see http://www.sfe.com.au/content/sfe/trading/con_specs.pdf). One Sydney business day preceding the second Friday of the relevant settlement.
   */
  RollConventionEnum_IMMAUD RollConventionEnum = iota + 1
  /**
   * The last trading day/expiration day of the Canadian Derivatives Exchange (Bourse de Montreal Inc) Three-month Canadian Bankers' Acceptance Futures (Ticker Symbol BAX). The second London banking day prior to the third Wednesday of the contract month. If the determined day is a Bourse or bank holiday in Montreal or Toronto, the last trading day shall be the previous bank business day. Per Canadian Derivatives Exchange BAX contract specification.
   */
  RollConventionEnum_IMMCAD RollConventionEnum = iota + 1
  /**
   * The last trading day of the Sydney Futures Exchange NZ 90 Day Bank Bill Futures contract (see http://www.sfe.com.au/content/sfe/trading/con_specs.pdf). The first Wednesday after the ninth day of the relevant settlement month.
   */
  RollConventionEnum_IMMNZD RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Monday.
   */
  RollConventionEnum_MON RollConventionEnum = iota + 1
  /**
   * The roll convention is not required. For example, in the case of a daily calculation frequency.
   */
  RollConventionEnum_NONE RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Saturday
   */
  RollConventionEnum_SAT RollConventionEnum = iota + 1
  /**
   * Sydney Futures Exchange 90-Day Bank Accepted Bill Futures Settlement Dates. The second Friday of the (delivery) month
   */
  RollConventionEnum_SFE RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Sunday
   */
  RollConventionEnum_SUN RollConventionEnum = iota + 1
  /**
   * 13-week and 26-week U.S. Treasury Bill Auction Dates. Each Monday except for U.S. (New York) holidays when it will occur on a Tuesday.
   */
  RollConventionEnum_TBILL RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Thursday
   */
  RollConventionEnum_THU RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Tuesday
   */
  RollConventionEnum_TUE RollConventionEnum = iota + 1
  /**
   * Rolling weekly on a Wednesday
   */
  RollConventionEnum_WED RollConventionEnum = iota + 1
  /**
   * Rolls on the 1st day of the month.
   */
  RollConventionEnum__1 RollConventionEnum = iota + 1
  /**
   * Rolls on the 10th day of the month.
   */
  RollConventionEnum__10 RollConventionEnum = iota + 1
  /**
   * Rolls on the 11th day of the month.
   */
  RollConventionEnum__11 RollConventionEnum = iota + 1
  /**
   * Rolls on the 12th day of the month.
   */
  RollConventionEnum__12 RollConventionEnum = iota + 1
  /**
   * Rolls on the 13th day of the month.
   */
  RollConventionEnum__13 RollConventionEnum = iota + 1
  /**
   * Rolls on the 14th day of the month.
   */
  RollConventionEnum__14 RollConventionEnum = iota + 1
  /**
   * Rolls on the 15th day of the month.
   */
  RollConventionEnum__15 RollConventionEnum = iota + 1
  /**
   * Rolls on the 16th day of the month.
   */
  RollConventionEnum__16 RollConventionEnum = iota + 1
  /**
   * Rolls on the 17th day of the month.
   */
  RollConventionEnum__17 RollConventionEnum = iota + 1
  /**
   * Rolls on the 18th day of the month.
   */
  RollConventionEnum__18 RollConventionEnum = iota + 1
  /**
   * Rolls on the 19th day of the month.
   */
  RollConventionEnum__19 RollConventionEnum = iota + 1
  /**
   * Rolls on the 2nd day of the month.
   */
  RollConventionEnum__2 RollConventionEnum = iota + 1
  /**
   * Rolls on the 20th day of the month.
   */
  RollConventionEnum__20 RollConventionEnum = iota + 1
  /**
   * Rolls on the 21st day of the month.
   */
  RollConventionEnum__21 RollConventionEnum = iota + 1
  /**
   * Rolls on the 22nd day of the month.
   */
  RollConventionEnum__22 RollConventionEnum = iota + 1
  /**
   * Rolls on the 23rd day of the month.
   */
  RollConventionEnum__23 RollConventionEnum = iota + 1
  /**
   * Rolls on the 24th day of the month.
   */
  RollConventionEnum__24 RollConventionEnum = iota + 1
  /**
   * Rolls on the 25th day of the month.
   */
  RollConventionEnum__25 RollConventionEnum = iota + 1
  /**
   * Rolls on the 26th day of the month.
   */
  RollConventionEnum__26 RollConventionEnum = iota + 1
  /**
   * Rolls on the 27th day of the month.
   */
  RollConventionEnum__27 RollConventionEnum = iota + 1
  /**
   * Rolls on the 28th day of the month.
   */
  RollConventionEnum__28 RollConventionEnum = iota + 1
  /**
   * Rolls on the 29th day of the month.
   */
  RollConventionEnum__29 RollConventionEnum = iota + 1
  /**
   * Rolls on the 3rd day of the month.
   */
  RollConventionEnum__3 RollConventionEnum = iota + 1
  /**
   * Rolls on the 30th day of the month.
   */
  RollConventionEnum__30 RollConventionEnum = iota + 1
  /**
   * Rolls on the 4th day of the month.
   */
  RollConventionEnum__4 RollConventionEnum = iota + 1
  /**
   * Rolls on the 5th day of the month.
   */
  RollConventionEnum__5 RollConventionEnum = iota + 1
  /**
   * Rolls on the 6th day of the month.
   */
  RollConventionEnum__6 RollConventionEnum = iota + 1
  /**
   * Rolls on the 7th day of the month.
   */
  RollConventionEnum__7 RollConventionEnum = iota + 1
  /**
   * Rolls on the 8th day of the month.
   */
  RollConventionEnum__8 RollConventionEnum = iota + 1
  /**
   * Rolls on the 9th day of the month.
   */
  RollConventionEnum__9 RollConventionEnum = iota + 1
  )    

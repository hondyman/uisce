/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the types of events found in the related Master Agreement that may lead to the early termination of the Master Agreement, including suspension of the affected party's rights to rehypothecate, transfer, or substitute collateral.
   */
  
  const (
  /**
   * Represents the Additional Termination Event(s) in the related Master Agreement.
   */
  AdditionalTerminationEventEnum_AS_APPLICABLE AdditionalTerminationEventEnum = iota + 1
  /**
   * Represents any bespoke Additional Termination Event(s) that are set out in the related Specified Condition clause.
   */
  AdditionalTerminationEventEnum_AS_SPECIFIED AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where a party fails to notify the other of a decline in Net Asset Value by a specified amount over a specified period of time.
   */
  AdditionalTerminationEventEnum_FAILURE_TO_NOTIFY_NAV AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where the investment advisor of a party ceases to act for the party.
   */
  AdditionalTerminationEventEnum_INVESTMENT_ADVISOR AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where a person deemed important to a party has departed.
   */
  AdditionalTerminationEventEnum_KEY_PERSONS AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event in which a partys Net Asset Value (NAV) has declined beyond a specified percentage or amount within a defined time period.
   */
  AdditionalTerminationEventEnum_NAV_DECLINE_TRIGGER AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event in which a partys Net Asset Value (NAV) falls below a predetermined absolute value, regardless of the timeframe over which the decline occurs. This value sets a minimum acceptable NAV threshold expressed as either a percentage or amount.
   */
  AdditionalTerminationEventEnum_NAV_FLOOR AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where a party delivers operative documents that are invalid, untrue or unenforceable.
   */
  AdditionalTerminationEventEnum_OPERATIVE_DOCS AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where Party 1 experiences a downgrade beyond a predetermined level, or has their credit rating withdrawn or suspended.
   */
  AdditionalTerminationEventEnum_RATINGS_DOWNGRADE_OR_WITHDRAWAL_PARTY_1 AdditionalTerminationEventEnum = iota + 1
  /**
   * Specifies an event where Party 2 experiences a downgrade beyond a predetermined level, or has their credit rating withdrawn or suspended.
   */
  AdditionalTerminationEventEnum_RATINGS_DOWNGRADE_OR_WITHDRAWAL_PARTY_2 AdditionalTerminationEventEnum = iota + 1
  )    

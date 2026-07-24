/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the types of events in the related Master Agreement that, when triggered, could temporarily or permanently suspend rights to rehypothecate, transfer or substitute collateral.
   */
  
  const (
  /**
   * Specifies events which could trigger rights to early termination.
   */
  CSASpecifiedOrAccessConditionEnum_ADDITIONAL_TERMINATION_EVENTS CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies an event where a party merges and the new entity becomes materially less creditworthy than the original.
   */
  CSASpecifiedOrAccessConditionEnum_CREDIT_EVENT_UPON_MERGER CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies an event beyond the control of one or both of the parties which prevents or renders it impossible to fulfill their obligations to the other party.
   */
  CSASpecifiedOrAccessConditionEnum_FORCE_MAJEURE_EVENT CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies an event where a party is unable to comply with its obligations under the Master Agreement because to do so would be unlawful.
   */
  CSASpecifiedOrAccessConditionEnum_ILLEGALITY CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies a potential event that could trigger rights to early termination.
   */
  CSASpecifiedOrAccessConditionEnum_POTENTIAL_TERMINATION_EVENTS CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies an event where a party experiences changes in tax laws incurring further tax liability.
   */
  CSASpecifiedOrAccessConditionEnum_TAX_EVENT CSASpecifiedOrAccessConditionEnum = iota + 1
  /**
   * Specifies an event where a party incurs tax liability due to a merger.
   */
  CSASpecifiedOrAccessConditionEnum_TAX_EVENT_UPON_MERGER CSASpecifiedOrAccessConditionEnum = iota + 1
  )    

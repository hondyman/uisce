/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines the enumerated values to specify the ancillary roles to the transaction. The product is agnostic to the actual parties involved in the transaction, with the party references abstracted away from the product definition and replaced by the AncillaryRoleEnum. The AncillaryRoleEnum can then be positioned in the product and the AncillaryParty type, which is positioned outside of the product definition, allows the AncillaryRoleEnum to be associated with an actual party reference.
   */
  
  const (
  /**
   * Specifies the party responsible for deciding the fallback rate.
   */
  AncillaryRoleEnum_CALCULATION_AGENT_FALLBACK AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party responsible for performing calculation agent duties as defined in the applicable product definition.
   */
  AncillaryRoleEnum_CALCULATION_AGENT_INDEPENDENT AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party responsible for performing calculation agent duties associated with an mandatory early termination.
   */
  AncillaryRoleEnum_CALCULATION_AGENT_MANDATORY_EARLY_TERMINATION AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party responsible for performing calculation agent duties associated with an optional early termination.
   */
  AncillaryRoleEnum_CALCULATION_AGENT_OPTIONAL_EARLY_TERMINATION AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party which determines additional disruption events.
   */
  AncillaryRoleEnum_DISRUPTION_EVENTS_DETERMINING_PARTY AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party to which notice of a cancelable provision exercise should be given.
   */
  AncillaryRoleEnum_EXERCISE_NOTICE_RECEIVER_PARTY_CANCELABLE_PROVISION AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party to which notice of a extendible provision exercise should be given.
   */
  AncillaryRoleEnum_EXERCISE_NOTICE_RECEIVER_PARTY_EXTENDIBLE_PROVISION AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party to which notice of a manual exercise should be given.
   */
  AncillaryRoleEnum_EXERCISE_NOTICE_RECEIVER_PARTY_MANUAL AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party to which notice of a optional early termination exercise should be given.
   */
  AncillaryRoleEnum_EXERCISE_NOTICE_RECEIVER_PARTY_OPTIONAL_EARLY_TERMINATION AncillaryRoleEnum = iota + 1
  /**
   * Specifies the party which determines if dividends are extraordinary in relation to normal levels.
   */
  AncillaryRoleEnum_EXTRAORDINARY_DIVIDENDS_PARTY AncillaryRoleEnum = iota + 1
  /**
   * Specifies the clearing organization (CCP, DCO) which the trade should be cleared.
   */
  AncillaryRoleEnum_PREDETERMINED_CLEARING_ORGANIZATION_PARTY AncillaryRoleEnum = iota + 1
  )    

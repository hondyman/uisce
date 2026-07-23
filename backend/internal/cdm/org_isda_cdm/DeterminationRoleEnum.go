/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines the enumerated values to specify the determination roles to the transaction. Such roles mostly address any determination that would be required when some Extraordinary Events would happen, for instance qualifying the effectiveness of such event, or when a calculation is required, etc. else any other kind of determination as need be.
   */
  
  const (
  /**
   * Specifies the party responsible for performing calculation agent duties as defined in the applicable product definition. As an indication, the role of Calculation Agent is key in regards of other roles, for mainly two reasons : first is that it is a fallback role i.e. whenever another role is not defined, then such role would be assumed by the Calculation Agent ; second is that related determination are not limited to Extraordinary Event per se i.e. any determinationr required in regards of Payout calculations for instance would fall on Calculation Agent, unless otherwise specified in Calculation Agent attached to Economic Terms.
   */
  DeterminationRoleEnum_CALCULATION_AGENT DeterminationRoleEnum = iota + 1
  /**
   * Specifies the party responsible for performing related HedgingParty duties as defined in the applicable product definition, notably in regards of particular Disruption Events or Early Termination Terms which may involve the calculation of a liquidation or compensation value amount.
   */
  DeterminationRoleEnum_DETERMINING_PARTY DeterminationRoleEnum = iota + 1
  /**
   * Specifies the party responsible for performing related HedgingParty duties as defined in the applicable product definition, notably in regards of particular Extraordinary Events or Price Determination Methods which involve hedging considerations.
   */
  DeterminationRoleEnum_HEDGING_PARTY DeterminationRoleEnum = iota + 1
  )    

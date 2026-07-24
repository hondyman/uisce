/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify how a calculation agent will be determined.
   */
  
  const (
  /**
   * The Calculation Agent is determined by reference to the relevant master agreement.
   */
  PartyDeterminationEnum_AS_SPECIFIED_IN_MASTER_AGREEMENT PartyDeterminationEnum = iota + 1
  /**
   * The Calculation Agent is determined by reference to the relevant standard terms supplement.
   */
  PartyDeterminationEnum_AS_SPECIFIED_IN_STANDARD_TERMS_SUPPLEMENT PartyDeterminationEnum = iota + 1
  /**
   * Both parties with joined rights to be a calculation agent.
   */
  PartyDeterminationEnum_BOTH PartyDeterminationEnum = iota + 1
  /**
   * The party that gives notice of exercise. Per 2000 ISDA Definitions, Section 11.1. Parties, paragraph (d).
   */
  PartyDeterminationEnum_EXERCISING_PARTY PartyDeterminationEnum = iota + 1
  /**
   * The party that is given notice of exercise. Per 2000 ISDA Definitions, Section 11.1. Parties, paragraph (e).
   */
  PartyDeterminationEnum_NON_EXERCISING_PARTY PartyDeterminationEnum = iota + 1
  )    

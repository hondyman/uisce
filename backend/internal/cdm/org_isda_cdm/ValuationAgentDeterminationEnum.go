/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies how the Valuation Agent is determined.
   */
  
  const (
  /**
   * There is a fixed party for disputes.
   */
  ValuationAgentDeterminationEnum_FIXED_PARTY_FOR_DISPUTES ValuationAgentDeterminationEnum = iota + 1
  /**
   * There is only a Sole Valuation Agent.
   */
  ValuationAgentDeterminationEnum_SOLE_VALUATION_AGENT ValuationAgentDeterminationEnum = iota + 1
  /**
   * Switch of Valuation Agent can occur upon Default.
   */
  ValuationAgentDeterminationEnum_SWITCH_UPON_DEFAULT ValuationAgentDeterminationEnum = iota + 1
  /**
   * Switch of Valuation Agent can occur upon failure to perform.
   */
  ValuationAgentDeterminationEnum_SWITCH_UPON_FAILURE_TO_PERFORM ValuationAgentDeterminationEnum = iota + 1
  )    

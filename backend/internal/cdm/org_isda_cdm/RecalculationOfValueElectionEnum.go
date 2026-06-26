/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Values to specify the procedure under which the market value of posted collateral will be recalculated.
   */
  
  const (
  /**
   * The parties agree to consult.
   */
  RecalculationOfValueElectionEnum_CONSULATION_PROCEDURE RecalculationOfValueElectionEnum = iota + 1
  /**
   * Description to be added.
   */
  RecalculationOfValueElectionEnum_NOT_APPLICABLE RecalculationOfValueElectionEnum = iota + 1
  /**
   * The procedures specified in an Other Regulatory CSA.
   */
  RecalculationOfValueElectionEnum_OTHER_REGULATORY_CSA_PROCEDURE RecalculationOfValueElectionEnum = iota + 1
  /**
   * Bespoke Recalculation of value terms are specified in the agreement.
   */
  RecalculationOfValueElectionEnum_SPECIFIED RecalculationOfValueElectionEnum = iota + 1
  )    

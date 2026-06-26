/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the form of the restructuring credit event that is applicable to the credit default swap.
   */
  
  const (
  /**
   * Restructuring (Section 4.7) and Modified Restructuring Maturity Limitation and Conditionally Transferable Obligation (2014 Definitions: Section 3.31, 2003 Definitions: 2.32) apply.
   */
  RestructuringEnum_MOD_MOD_R RestructuringEnum = iota + 1
  /**
   * Restructuring (Section 4.7) and Restructuring Maturity Limitation and Fully Transferable Obligation (2014 Definitions: Section 3.31, 2003 Definitions: 2.32) apply.
   */
  RestructuringEnum_MOD_R RestructuringEnum = iota + 1
  /**
   * Restructuring as defined in the applicable ISDA Credit Derivatives Definitions. (2003 or 2014).
   */
  RestructuringEnum_R RestructuringEnum = iota + 1
  )    

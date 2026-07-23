/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the type of a resource (e.g. document).
   */
  
  const (
  /**
   * Document describing the legal terms of a transaction.
   */
  ResourceTypeEnum_CONFIRMATION ResourceTypeEnum = iota + 1
  /**
   * Document providing supplemental material economic terms to the FpML data representation. The initial intended usage is to fulfill the CFTC Part 45 rule requirement to report ‘Any other terms(s) of the swap matched or affirmed by the counterparties in verifying the swap’ when the reporting is done via the generic FpML representation.
   */
  ResourceTypeEnum_SUPPLEMENTAL_MATERIAL_ECONOMIC_TERMS ResourceTypeEnum = iota + 1
  /**
   * Document describing the economic characteristics of a transaction.
   */
  ResourceTypeEnum_TERM_SHEET ResourceTypeEnum = iota + 1
  )    

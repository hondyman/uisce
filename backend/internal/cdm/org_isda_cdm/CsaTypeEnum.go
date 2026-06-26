/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * How is the Creadit Support Annex defined for this transaction as defined in the 2021 ISDA Definitions, section 18.2.1 
   */
  
  const (
  /**
   * Thre is an existing Credit Support Annex
   */
  CsaTypeEnum_EXISTING_CSA CsaTypeEnum = iota + 1
  /**
   * There is no CSA applicable
   */
  CsaTypeEnum_NO_CSA CsaTypeEnum = iota + 1
  /**
   * There is a bilateral Credit Support Annex specific to the transaction
   */
  CsaTypeEnum_REFERENCE_VMCSA CsaTypeEnum = iota + 1
  )    

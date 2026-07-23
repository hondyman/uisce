/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Values to specify the SIMM normalized exception approaches.
   */
  
  const (
  /**
   * The ISDA Standard Initial Margin Model exception is applicable as a Fallback to Mandatory Method.
   */
  SimmExceptionApplicableEnum_FALL_BACK_TO_MANDATORY_METHOD SimmExceptionApplicableEnum = iota + 1
  /**
   * The ISDA Standard Initial Margin Model exception is applicable as a Mandatory Method.
   */
  SimmExceptionApplicableEnum_MANDATORY_METHOD SimmExceptionApplicableEnum = iota + 1
  /**
   * An alternative approach is described in the document.
   */
  SimmExceptionApplicableEnum_OTHER_METHOD SimmExceptionApplicableEnum = iota + 1
  )    

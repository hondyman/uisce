/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the specified entity terms for the Event of Default or Termination Event specified.
   */
  
  const (
  /**
   * Any Affiliate is a Specified Entity.
   */
  SpecifiedEntityTermsEnum_ANY_AFFILIATE SpecifiedEntityTermsEnum = iota + 1
  /**
   * Any Material Subsidiary.
   */
  SpecifiedEntityTermsEnum_MATERIAL_SUBSIDIARY SpecifiedEntityTermsEnum = iota + 1
  /**
   * The Specified Entity is provided.
   */
  SpecifiedEntityTermsEnum_NAMED_SPECIFIED_ENTITY SpecifiedEntityTermsEnum = iota + 1
  /**
   * No Specified Entity is provided
   */
  SpecifiedEntityTermsEnum_NONE SpecifiedEntityTermsEnum = iota + 1
  /**
   * Non standard Specified Entity terms are provided.
   */
  SpecifiedEntityTermsEnum_OTHER_SPECIFIED_ENTITY SpecifiedEntityTermsEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify the characteristics of the rating if there are several agency issue ratings but not equivalent, reference will be made to label characteristics of the rating such as the lowest/highest available.
   */
  
  const (
  /**
   * Denotes the average credit notation if several notations are listed.
   */
  CreditNotationMismatchResolutionEnum_AVERAGE CreditNotationMismatchResolutionEnum = iota + 1
  /**
   * Denotes the highest credit notation if several notations are listed.
   */
  CreditNotationMismatchResolutionEnum_HIGHEST CreditNotationMismatchResolutionEnum = iota + 1
  /**
   * Denotes the lowest credit notation if several notations are listed.
   */
  CreditNotationMismatchResolutionEnum_LOWEST CreditNotationMismatchResolutionEnum = iota + 1
  /**
   * Utilised where bespoke language represents the label characteristics of the rating.
   */
  CreditNotationMismatchResolutionEnum_OTHER CreditNotationMismatchResolutionEnum = iota + 1
  /**
   * Denotes that a credit notation issued from a defined reference agency is used if several notations are listed.
   */
  CreditNotationMismatchResolutionEnum_REFERENCE_AGENCY CreditNotationMismatchResolutionEnum = iota + 1
  /**
   * Denotes the second best credit notation if several notations are listed.
   */
  CreditNotationMismatchResolutionEnum_SECOND_BEST CreditNotationMismatchResolutionEnum = iota + 1
  )    

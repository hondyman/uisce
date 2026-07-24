/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * What calculation type is required, averaging or compounding. This enumeration is used to represent the definitions of modular calculated rates as described in the 2021 ISDA Definitions, section 7.
   */
  
  const (
  /**
   * Averaging, i.e. arithmetic averaging.
   */
  CalculationMethodEnum_AVERAGING CalculationMethodEnum = iota + 1
  /**
   * A rate based on an index that is computed by a rate administrator.  The user is responsible for backing out the rate by applying a simple formula.
   */
  CalculationMethodEnum_COMPOUNDED_INDEX CalculationMethodEnum = iota + 1
  /**
   * Compounding, i.e. geometric averaging following an ISDA defined formula.
   */
  CalculationMethodEnum_COMPOUNDING CalculationMethodEnum = iota + 1
  )    

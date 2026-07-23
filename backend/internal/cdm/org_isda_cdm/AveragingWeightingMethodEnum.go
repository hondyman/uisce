/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the method of calculation to be used when averaging rates. Per ISDA 2000 Definitions, Section 6.2. Certain Definitions Relating to Floating Amounts.
   */
  
  const (
  /**
   * The arithmetic mean of the relevant rates for each reset date.
   */
  AveragingWeightingMethodEnum_UNWEIGHTED AveragingWeightingMethodEnum = iota + 1
  /**
   * The arithmetic mean of the relevant rates in effect for each day in a calculation period calculated by multiplying each relevant rate by the number of days such relevant rate is in effect, determining the sum of such products and dividing such sum by the number of days in the calculation period.
   */
  AveragingWeightingMethodEnum_WEIGHTED AveragingWeightingMethodEnum = iota + 1
  )    

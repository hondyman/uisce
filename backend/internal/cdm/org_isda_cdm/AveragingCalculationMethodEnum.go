/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies enumerations for the type of averaging calculation.
   */
  
  const (
  /**
   * Refers to the calculation of an average by taking the sum of observations divided by the count of observations.
   */
  AveragingCalculationMethodEnum_ARITHMETIC AveragingCalculationMethodEnum = iota + 1
  /**
   * Refers to the calculation of an average by taking the nth root of the product of n observations.
   */
  AveragingCalculationMethodEnum_GEOMETRIC AveragingCalculationMethodEnum = iota + 1
  /**
   * Refers to the calculation of an average by taking the reciprocal of the arithmetic mean of the reciprocals of the observations.
   */
  AveragingCalculationMethodEnum_HARMONIC AveragingCalculationMethodEnum = iota + 1
  )    

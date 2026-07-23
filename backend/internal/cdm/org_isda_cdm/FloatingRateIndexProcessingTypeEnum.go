/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * This enumeration provides guidance on how to process a given floating rate index.  It's based on the ISDA Floating Rate Index information, but transforms it into the specific categories needed for calculation 
   */
  
  const (
  /**
   * A published index calculated using compounding; the implied rate must be backed out.
   */
  FloatingRateIndexProcessingTypeEnum_COMPOUND_INDEX FloatingRateIndexProcessingTypeEnum = iota + 1
  /**
   * These are calculated by the calculation agent based on deal-specific parameters (e.g. lookback compound based on an RFR).
   */
  FloatingRateIndexProcessingTypeEnum_MODULAR FloatingRateIndexProcessingTypeEnum = iota + 1
  /**
   * These are calculated by the calculation agent based on a standard OIS FRO definition.
   */
  FloatingRateIndexProcessingTypeEnum_OIS FloatingRateIndexProcessingTypeEnum = iota + 1
  /**
   * These are calculated by the calculation agent based on a standard overnight averaging FRO definition.
   */
  FloatingRateIndexProcessingTypeEnum_OVERNIGHT_AVG FloatingRateIndexProcessingTypeEnum = iota + 1
  /**
   * These must be looked up using a manual process
   */
  FloatingRateIndexProcessingTypeEnum_REF_BANKS FloatingRateIndexProcessingTypeEnum = iota + 1
  /**
   * These values are just looked up from the screen and applied.
   */
  FloatingRateIndexProcessingTypeEnum_SCREEN FloatingRateIndexProcessingTypeEnum = iota + 1
  )    

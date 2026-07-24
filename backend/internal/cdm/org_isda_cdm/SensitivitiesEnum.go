/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Values to specify the methodology according to which sensitivities to (i) equity indices, funds and ETFs, and (ii) commodity indices are computed.
   */
  
  const (
  /**
   * The parties agree that in respect of the relevant sensitivities, the delta is allocated back to individual constituents.
   */
  SensitivitiesEnum_ALTERNATIVE SensitivitiesEnum = iota + 1
  /**
   * The relevant sensitivities are addressed by the standard preferred approach where the entire delta is put into the applicable asset class/category.
   */
  SensitivitiesEnum_STANDARD SensitivitiesEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the relevant settled entity matrix source.
   */
  
  const (
  /**
   * The Relevant Settled Entity Matrix shall be the list agreed for this purpose by the parties. The list is not included as part of the electronic confirmation.
   */
  SettledEntityMatrixSourceEnum_CONFIRMATION_ANNEX SettledEntityMatrixSourceEnum = iota + 1
  /**
   * The term is not applicable.
   */
  SettledEntityMatrixSourceEnum_NOT_APPLICABLE SettledEntityMatrixSourceEnum = iota + 1
  /**
   * The Settled Entity Matrix published by the Index Publisher.
   */
  SettledEntityMatrixSourceEnum_PUBLISHER SettledEntityMatrixSourceEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * This indicator defines which type of assets (cash or securities) is specified to apply as margin to the repo transaction.
   */
  
  const (
  /**
   * When the margin type is Cash, the margin factor is applied to the cash value of the transaction.
   */
  MarginTypeEnum_CASH MarginTypeEnum = iota + 1
  /**
   * When the margin type is Instrument, the margin factor is applied to the instrument value for the transaction. In the 'instrument' case, the haircut would be applied to the securities.
   */
  MarginTypeEnum_INSTRUMENT MarginTypeEnum = iota + 1
  )    

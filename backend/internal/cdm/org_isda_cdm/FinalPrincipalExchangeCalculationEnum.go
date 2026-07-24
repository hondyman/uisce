/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * To be specified only for products that embed a redemption payment.
   */
  
  const (
  /**
   * If Floored is set then Principal Exchange takes the form: Notional Amount * Max(1, Index Final/ Index Base).
   */
  FinalPrincipalExchangeCalculationEnum_FLOORED FinalPrincipalExchangeCalculationEnum = iota + 1
  /**
   * If NonFloored is set then the Principal Exchange takes the form: Notional Amount * Index Final / Index Base.
   */
  FinalPrincipalExchangeCalculationEnum_NON_FLOORED FinalPrincipalExchangeCalculationEnum = iota + 1
  )    

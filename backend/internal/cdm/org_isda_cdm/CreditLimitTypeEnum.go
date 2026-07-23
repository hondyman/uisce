/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to qualify the type of credit limits.
   */
  
  const (
  /**
   * The type of credit line expressed in CS01. The sensitivity with respect to changes in the CDS spread.
   */
  CreditLimitTypeEnum_CS01 CreditLimitTypeEnum = iota + 1
  /**
   * The type of credit line expressed in DV01. The dollar value of a one basis point decrease in interest rates. It shows the change in a bond's price compared to a decrease in the bond's yield.
   */
  CreditLimitTypeEnum_DV01 CreditLimitTypeEnum = iota + 1
  /**
   * The type of credit line expressed in Initial Margin value.
   */
  CreditLimitTypeEnum_IM CreditLimitTypeEnum = iota + 1
  /**
   * The type of credit line expressed as a Net Present Value.
   */
  CreditLimitTypeEnum_NPV CreditLimitTypeEnum = iota + 1
  /**
   * The type of credit line expressed in Notional amount.
   */
  CreditLimitTypeEnum_NOTIONAL CreditLimitTypeEnum = iota + 1
  /**
   * The type of credit line expressed in PV01. The value of a one dollar or one basis point annuity.
   */
  CreditLimitTypeEnum_PV01 CreditLimitTypeEnum = iota + 1
  )    

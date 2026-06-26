/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list that identifies the type of debt.
   */
  
  const (
  /**
   * Identifies a debt instrument that can be converted into common shares.
   */
  DebtClassEnum_CONVERTIBLE DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument that can be converted primarily at the election of the holder into common shares of the Issuer.
   */
  DebtClassEnum_HOLDER_CONVERTIBLE DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument that can be converted primarily at the election of the holder into common shares of a party other than the Issuer.
   */
  DebtClassEnum_HOLDER_EXCHANGEABLE DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument that can be converted at the election of the Issuer into common shares of the Issuer.  Also known as reverse convertible.
   */
  DebtClassEnum_ISSUER_CONVERTIBLE DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument that can be converted at the election of the Issuer into common shares of a party other than the Issuer.  Also known as reverse exchangeable.
   */
  DebtClassEnum_ISSUER_EXCHANGEABLE DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument as one issued by financial institutions to count towards regulatory capital, including term and perpetual subordinated debt, contingently convertible and others.  Excludes preferred share capital.
   */
  DebtClassEnum_REG_CAP DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument athat has non-standard interest or principal features, with full recourse to the issuer.
   */
  DebtClassEnum_STRUCTURED DebtClassEnum = iota + 1
  /**
   * Identifies a debt instrument that has a periodic coupon, a defined maturity, and is not backed by any specific asset. The seniority and the structure of the income and principal payments can optionally be defined in DebtType.DebtEconomics.
   */
  DebtClassEnum_VANILLA DebtClassEnum = iota + 1
  )    

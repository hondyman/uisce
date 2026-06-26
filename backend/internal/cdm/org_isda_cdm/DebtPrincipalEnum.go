/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list that specifies the general rule for repayment of principal.
   */
  
  const (
  /**
   * Denotes that the principal on the debt is paid down regularly, along with its interest expense over the life of the debt instrument.  Includes amortising instruments with a bullet balance repayment at maturity.
   */
  DebtPrincipalEnum_AMORTISING DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the principal is paid all at once on maturity of the debt insrument. Bullet debt instruments cannot be redeemed early by an issuer, which means they are non-callable.
   */
  DebtPrincipalEnum_BULLET DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the principal on the debt can be repaid early, in whole or in part, at the option of the issuer.
   */
  DebtPrincipalEnum_CALLABLE DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the  principal on the debt is calculated with reference to one or more price or other indices (other than inflation rates).
   */
  DebtPrincipalEnum_INDEX_LINKED DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the principal on the debt is calculated with reference to one or more specified inflation rates.
   */
  DebtPrincipalEnum_INFLATION_LINKED DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the  principal on the debt is calculated with reference to other underlyings (not being floating interest rates, inflation rates or indices) or with a non-linear relationship to floating interest rates, inflation rates or indices.
   */
  DebtPrincipalEnum_OTHER_STRUCTURED DebtPrincipalEnum = iota + 1
  /**
   * Denotes a stripped bond representing only the principal component.
   */
  DebtPrincipalEnum_PRINCIPAL_ONLY DebtPrincipalEnum = iota + 1
  /**
   * Denotes that the principal on the debt can be repaid early, in whole or in part, at the option of the holder.
   */
  DebtPrincipalEnum_PUTTABLE DebtPrincipalEnum = iota + 1
  )    

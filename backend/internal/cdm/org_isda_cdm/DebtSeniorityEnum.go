/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the order of repayment in the event of a sale or bankruptcy of the issuer or a related party (eg guarantor).
   */
  
  const (
  /**
   * Denotes debt which is secured over assets of the issuer or a related party (eg guarantor).
   */
  DebtSeniorityEnum_SECURED DebtSeniorityEnum = iota + 1
  /**
   * Denotes debt  which ranks pari passu with all other unsecured creditors of the issuer.
   */
  DebtSeniorityEnum_SENIOR DebtSeniorityEnum = iota + 1
  /**
   * Denotes debt  owed to an unsecured creditor that in the event of a liquidation can only be paid after the claims of secured and senior creditors have been met.
   */
  DebtSeniorityEnum_SUBORDINATED DebtSeniorityEnum = iota + 1
  )    

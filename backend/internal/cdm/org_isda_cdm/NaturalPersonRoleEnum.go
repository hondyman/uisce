/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values for the natural person's role.
   */
  
  const (
  /**
   * The person who arranged with a client to execute the trade.
   */
  NaturalPersonRoleEnum_BROKER NaturalPersonRoleEnum = iota + 1
  /**
   * Acquirer of the legal title to the financial instrument.
   */
  NaturalPersonRoleEnum_BUYER NaturalPersonRoleEnum = iota + 1
  /**
   * The party or person with legal responsibility for authorization of the execution of the transaction.
   */
  NaturalPersonRoleEnum_DECISION_MAKER NaturalPersonRoleEnum = iota + 1
  /**
   * Person within the firm who is responsible for execution of the transaction.
   */
  NaturalPersonRoleEnum_EXECUTION_WITHIN_FIRM NaturalPersonRoleEnum = iota + 1
  /**
   * Person who is responsible for making the investment decision.
   */
  NaturalPersonRoleEnum_INVESTMENT_DECISION_MAKER NaturalPersonRoleEnum = iota + 1
  /**
   * Seller of the legal title to the financial instrument.
   */
  NaturalPersonRoleEnum_SELLER NaturalPersonRoleEnum = iota + 1
  /**
   * The person who executed the trade.
   */
  NaturalPersonRoleEnum_TRADER NaturalPersonRoleEnum = iota + 1
  )    

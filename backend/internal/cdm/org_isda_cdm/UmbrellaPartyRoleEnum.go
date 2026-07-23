/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  
  const (
  /**
   * Represents a Contractual Party who has authority to negotiate, execute and deliver confirmations on behalf of each unique party to the agreement that is linked to the Agent.
   */
  UmbrellaPartyRoleEnum_AGENT UmbrellaPartyRoleEnum = iota + 1
  /**
   * Represents a Contractual Party who has been authorised to act as a centralised authority empowered to negotiate, execute and manage transactions on behalf of multiple affiliated funds or accounts.
   */
  UmbrellaPartyRoleEnum_INVESTMENT_MANAGER UmbrellaPartyRoleEnum = iota + 1
  /**
   * Represents a Contractual Party that enters into and assumes direct responsibility for transactions.
   */
  UmbrellaPartyRoleEnum_PRINCIPAL UmbrellaPartyRoleEnum = iota + 1
  /**
   * Represents a distinct trading strategy, portfolio, or sub account managed within a broader legal entity or fund. It is not a legal party to the agreement.
   */
  UmbrellaPartyRoleEnum_SLEEVE UmbrellaPartyRoleEnum = iota + 1
  /**
   * Represents an individual trading party, fund, portfolio or managed account associated to a principal, Investment Manager or Agent Contractual Party.
   */
  UmbrellaPartyRoleEnum_SUB_ACCOUNT UmbrellaPartyRoleEnum = iota + 1
  )    

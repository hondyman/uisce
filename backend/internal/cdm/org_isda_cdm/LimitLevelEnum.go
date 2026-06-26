/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to specify the level at which the limit is set: customer business, proprietary business or account level. This is part of the CME specification for clearing credit limits, although not specified as a set of enumerated values as part of the clearing confirmation specification.
   */
  
  const (
  /**
   * The limit is set in relation to the proprietary business undertaken by the clearing counterparty.
   */
  LimitLevelEnum_ACCOUNT LimitLevelEnum = iota + 1
  /**
   * The limit is set in relation to the customer business undertaken by the clearing counterparty.
   */
  LimitLevelEnum_CUSTOMER LimitLevelEnum = iota + 1
  /**
   * The limit is set at the account level in relation to the clearing counterparty.
   */
  LimitLevelEnum_HOUSE LimitLevelEnum = iota + 1
  )    

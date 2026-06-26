/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify how a contract has been executed, e.g. electronically, verbally, ...
   */
  
  const (
  /**
   * Execution via electronic execution facility, derivatives contract market, or other electronic message such as an instant message.
   */
  ExecutionTypeEnum_ELECTRONIC ExecutionTypeEnum = iota + 1
  /**
   * Bilateral execution between counterparties not pursuant to the rules of a SEF or DCM.
   */
  ExecutionTypeEnum_OFF_FACILITY ExecutionTypeEnum = iota + 1
  /**
   * Execution via a platform that may or may not be covered by a regulatory defintion. OnVenue is intended to distinguish trades executed on a trading platform from those executed via phone, email or messaging apps. The role and details of the venue are included in the party attribute of the trade. The general rule is that if the parties utilitzed the services of the platform to execute the trade then it would be considered OnVenue.
   */
  ExecutionTypeEnum_ON_VENUE ExecutionTypeEnum = iota + 1
  )    

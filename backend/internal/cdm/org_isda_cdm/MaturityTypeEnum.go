/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify the Maturity.
   */
  
  const (
  /**
   * Denotes a period from issuance date until now.
   */
  MaturityTypeEnum_FROM_ISSUANCE MaturityTypeEnum = iota + 1
  /**
   * Denotes a period from issuance until maturity date.
   */
  MaturityTypeEnum_ORIGINAL_MATURITY MaturityTypeEnum = iota + 1
  /**
   * Denotes a period from now until maturity date.
   */
  MaturityTypeEnum_REMAINING_MATURITY MaturityTypeEnum = iota + 1
  )    

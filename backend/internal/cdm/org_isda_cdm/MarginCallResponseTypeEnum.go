/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumeration values to define the response type to a margin call.
   */
  
  const (
  /**
   * Specifies a 'Full Agreement' to Margin Call.
   */
  MarginCallResponseTypeEnum_AGREEIN_FULL MarginCallResponseTypeEnum = iota + 1
  /**
   * Specifies a 'Full Dispute' to a Margin call.
   */
  MarginCallResponseTypeEnum_DISPUTE MarginCallResponseTypeEnum = iota + 1
  /**
   * Specifies a 'Partial agreement' to Margin Call.
   */
  MarginCallResponseTypeEnum_PARTIALLY_AGREE MarginCallResponseTypeEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Enumeration to describe the type of AvailableInventory
   */
  
  const (
  /**
   * Where a lender is broadcasting the securities that they have available to lend
   */
  AvailableInventoryTypeEnum_AVAILABLE_TO_LEND AvailableInventoryTypeEnum = iota + 1
  /**
   * Where a party is asking a lender if they have specific securities available for them to borrow
   */
  AvailableInventoryTypeEnum_REQUEST_TO_BORROW AvailableInventoryTypeEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  
  const (
  /**
   * Describes the reference contract as the one that pertains to the month-year of the calculation period. If used, the nearby count is expected to be 0.
   */
  DeliveryNearbyTypeEnum_CALCULATION_PERIOD DeliveryNearbyTypeEnum = iota + 1
  /**
   * Specifies that the reference delivery date of the underlying Commodity shall be the expiration date of the futures contract in the nth nearby month.
   */
  DeliveryNearbyTypeEnum_NEARBY_MONTH DeliveryNearbyTypeEnum = iota + 1
  /**
   * Specifies that the reference delivery date of the underlying Commodity shall be the expiration date of the futures contract in the nth nearby week.
   */
  DeliveryNearbyTypeEnum_NEARBY_WEEK DeliveryNearbyTypeEnum = iota + 1
  )    

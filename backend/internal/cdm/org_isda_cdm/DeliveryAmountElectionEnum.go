/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the application of Interest Amount with respect to the Delivery Amount through standard language.
   */
  
  const (
  /**
   * The delivery includes both `Transfer on last Local Business Day` and `Transfer a Delivery Amount (IM) consisting of cash on any Local Business Day.`
   */
  DeliveryAmountElectionEnum_LAST_AND_ANY_LOCAL_BUSINESS_DAY DeliveryAmountElectionEnum = iota + 1
  /**
   * The delivery only includes `Transfer on last Local Business Day.
   */
  DeliveryAmountElectionEnum_LAST_LOCAL_BUSINESS_DAY DeliveryAmountElectionEnum = iota + 1
  )    

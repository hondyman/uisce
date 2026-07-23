/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines the enumerated values to specify the nature of a location identifier.
   */
  
  const (
  /**
   * The hub code of the buyer.
   */
  CommodityLocationIdentifierTypeEnum_BUYER_HUB CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * The physical or virtual point at which the commodity will be delivered.
   */
  CommodityLocationIdentifierTypeEnum_DELIVERY_POINT CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * The zone covering potential delivery points for the commodity
   */
  CommodityLocationIdentifierTypeEnum_DELIVERY_ZONE CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * The physical or virtual point at which the commodity enters a transportation system.
   */
  CommodityLocationIdentifierTypeEnum_ENTRY_POINT CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * Identification of the border(s) or border point(s) of a transportation contract.
   */
  CommodityLocationIdentifierTypeEnum_INTERCONNECTION_POINT CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * The hub code of the seller.
   */
  CommodityLocationIdentifierTypeEnum_SELLER_HUB CommodityLocationIdentifierTypeEnum = iota + 1
  /**
   * The physical or virtual point at which the commodity is withdrawn from a transportation system.
   */
  CommodityLocationIdentifierTypeEnum_WITHDRAWAL_POINT CommodityLocationIdentifierTypeEnum = iota + 1
  )    

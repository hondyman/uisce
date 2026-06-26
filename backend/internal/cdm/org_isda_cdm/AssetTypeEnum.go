/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify the asset type.
   */
  
  const (
  /**
   * Indentifies cash in a currency form.
   */
  AssetTypeEnum_CASH AssetTypeEnum = iota + 1
  /**
   * Indentifies basic good used in commerce that is interchangeable with other goods of the same type.
   */
  AssetTypeEnum_COMMODITY AssetTypeEnum = iota + 1
  /**
   * Indentifies other asset types.
   */
  AssetTypeEnum_OTHER AssetTypeEnum = iota + 1
  /**
   * Indentifies negotiable financial instrument of monetary value with an issue ownership position.
   */
  AssetTypeEnum_SECURITY AssetTypeEnum = iota + 1
  )    

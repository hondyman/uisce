/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumeration values to specify the margin type in relation to bilateral or regulatory obligation.
   */
  
  const (
  /**
   * Indicates Non Regulatory Initial margin or independent amount
   */
  RegMarginTypeEnum_NON_REG_IM RegMarginTypeEnum = iota + 1
  /**
   * Indicates Regulatory Initial Margin
   */
  RegMarginTypeEnum_REG_IM RegMarginTypeEnum = iota + 1
  /**
   * Indicates Variation Margin
   */
  RegMarginTypeEnum_VM RegMarginTypeEnum = iota + 1
  )    

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify the type of Equity.
   */
  
  const (
  /**
   * Identifies an Equity of Convertible Preference, a stock which can be converted into a predetermined number of common shares and holds priority to receive capital return in event of issuer liquidation.
   */
  EquityTypeEnum_CONVERTIBLE_PREFERENCE EquityTypeEnum = iota + 1
  /**
   * Identifies a negotiable depositary receipt certificate issued by a bank representing shares in a foreign company traded on a local stock exchange.
   */
  EquityTypeEnum_DEPOSITARY_RECEIPT EquityTypeEnum = iota + 1
  /**
   * Identifies an Equity of Non-Convertible Preference, Shares which hold priority to receive capital return in event of issuer liquidation.
   */
  EquityTypeEnum_NON_CONVERTIBLE_PREFERENCE EquityTypeEnum = iota + 1
  /**
   * Identifies an Equity of Common stocks and shares.
   */
  EquityTypeEnum_ORDINARY EquityTypeEnum = iota + 1
  )    

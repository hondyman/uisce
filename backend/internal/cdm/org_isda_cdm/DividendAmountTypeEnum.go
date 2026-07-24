/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify whether the dividend is paid with respect to the Dividend Period.
   */
  
  const (
  /**
   * The Amount is determined as provided in the relevant Master Confirmation.
   */
  DividendAmountTypeEnum_AS_SPECIFIED_IN_MASTER_CONFIRMATION DividendAmountTypeEnum = iota + 1
  /**
   * The ex-date for a dividend occurs during a dividend period.
   */
  DividendAmountTypeEnum_EX_AMOUNT DividendAmountTypeEnum = iota + 1
  /**
   * The payment date for a dividend occurs during a dividend period.
   */
  DividendAmountTypeEnum_PAID_AMOUNT DividendAmountTypeEnum = iota + 1
  /**
   * The record date for a dividend occurs during a dividend period.
   */
  DividendAmountTypeEnum_RECORD_AMOUNT DividendAmountTypeEnum = iota + 1
  )    

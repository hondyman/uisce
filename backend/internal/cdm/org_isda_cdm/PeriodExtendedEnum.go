/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify a time period containing the additional value of Term.
   */
  
  const (
  /**
   * CalculationPeriod - the period corresponds to the calculation period   For example, used in the Commodity Markets to indicate that a reference contract is the one that corresponds to the period of the calculation period.
   */
  PeriodExtendedEnum_C PeriodExtendedEnum = iota + 1
  /**
   * Day
   */
  PeriodExtendedEnum_D PeriodExtendedEnum = iota + 1
  /**
   * Hour
   */
  PeriodExtendedEnum_H PeriodExtendedEnum = iota + 1
  /**
   * Month
   */
  PeriodExtendedEnum_M PeriodExtendedEnum = iota + 1
  /**
   * Term. The period commencing on the effective date and ending on the termination date. The T period always appears in association with periodMultiplier = 1, and the notation is intended for use in contexts where the interval thus qualified (e.g. accrual period, payment period, reset period, ...) spans the entire term of the trade.
   */
  PeriodExtendedEnum_T PeriodExtendedEnum = iota + 1
  /**
   * Week
   */
  PeriodExtendedEnum_W PeriodExtendedEnum = iota + 1
  /**
   * Year
   */
  PeriodExtendedEnum_Y PeriodExtendedEnum = iota + 1
  )    

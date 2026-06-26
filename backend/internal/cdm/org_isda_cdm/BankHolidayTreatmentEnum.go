/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines whether the bank holidays are treated as weekdays or weekends in terms of delivery profile in the context of commodity products, in particular those with peak or off-peak delivery profiles.
   */
  
  const (
  /**
   * Bank holidays treated as weekdays.
   */
  BankHolidayTreatmentEnum_AS_WEEKDAY BankHolidayTreatmentEnum = iota + 1
  /**
   * Bank holidays treated as weekends.
   */
  BankHolidayTreatmentEnum_AS_WEEKEND BankHolidayTreatmentEnum = iota + 1
  )    
